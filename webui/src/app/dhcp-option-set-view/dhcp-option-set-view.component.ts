import { Component, Input, OnInit } from '@angular/core'
import { TreeNode } from 'primeng/api'
import { DHCPOption } from '../backend/model/dHCPOption'
import { DhcpOptionsService } from '../dhcp-options.service'
import { IPType } from '../iptype'

/**
 * A node of the displayed option holding its basic description.
 */
export interface OptionNode {
    /**
     * Indicates if the option is always sent to a DHCP client, regardless
     * if the client has requested the option.
     */
    alwaysSend?: boolean
    /**
     * Option code.
     */
    code: number
    /**
     * Option universe (IPv4 or IPv6).
     */
    universe: IPType
    /**
     * Name of the configuration level at which the option has been configured.
     */
    level: string
}

/**
 * A node of the displayed option comprising information for a single
 * option field.
 */
export interface OptionFieldNode {
    fieldType: string
    value: string
}

/**
 * A component displaying configured DHCP options as a tree.
 */
@Component({
    selector: 'app-dhcp-option-set-view',
    templateUrl: './dhcp-option-set-view.component.html',
    styleUrls: ['./dhcp-option-set-view.component.sass'],
})
export class DhcpOptionSetViewComponent implements OnInit {
    /**
     * An input parameter holding an array of DHCP options associated with
     * a particular daemon and a host, subnet etc.
     *
     * The first array length must be equal to the length of the @link levels
     * array. It holds option sets at each configuration level. The second array
     * holds options defined at the particular level.
     */
    @Input() options: DHCPOption[][]

    /**
     * An input parameter holding an array of the configuration levels at
     * which the options have been configured.
     *
     * The typical sets are:
     * - subnet, shared network, global
     * - subnet, global
     * - host
     *
     * The number of levels must be equal to the length of the @link options array.
     */
    @Input() levels: string[]

    /**
     * Collection of the converted options into the nodes that can be
     * displayed as a tree.
     *
     * The array holds the options specified at each configuration level.
     */
    optionNodes: Array<Array<TreeNode<OptionNode | OptionFieldNode>>> = []

    /**
     * A flat collection of the converted options into the nodes that can be
     * displayed as a tree.
     *
     * This collection holds options from all inheritance levels combined into a single
     * tree. The options from the lower configuration levels take precedence over the
     * same options specified at the higher configuration levels.
     */
    combinedOptionNodes: Array<TreeNode<OptionNode | OptionFieldNode>> = []

    /**
     * A collection of currently displayed options.
     *
     * This collection points to one of the @link combinedOptionNodes or @link optionNodes,
     * depending on the @link currentLevelOnlyMode state.
     */
    displayedOptionNodes: Array<TreeNode<OptionNode | OptionFieldNode>> = []

    /**
     * A flag indicating whether all (combined) options should be displayed or the ones
     * from the lowest inheritance level.
     */
    currentLevelOnlyMode: boolean = false

    /**
     * Constructor.
     */
    constructor(public optionsService: DhcpOptionsService) {}

    /**
     * A component lifecycle hook executed when the component is initialized.
     *
     * It converts input DHCP options into the nodes tree that can be displayed.
     */
    ngOnInit(): void {
        for (let i = 0; i < this.options?.length; i++) {
            this.optionNodes.push(this.convertOptionsToNodes(this.options[i], this.levels[i]))
        }

        for (let treeOptionNodes of this.optionNodes) {
            for (let treeOptionNode of treeOptionNodes)
                if (treeOptionNode.type === 'option') {
                    const optionNode = treeOptionNode.data as OptionNode
                    if (
                        this.combinedOptionNodes
                            .map((n) => n.data as OptionNode)
                            // Check if there is another node with the same
                            // code but different level. We allow to have the
                            // same option code at the same level to display
                            // the duplicated options, e.g., for host
                            // reservations.
                            .some((d) => d.code === optionNode.code && d.level !== optionNode.level)
                    ) {
                        continue
                    }
                    this.combinedOptionNodes.push(treeOptionNode)
                }
        }
        this.displayedOptionNodes = this.combinedOptionNodes
    }

    /**
     * Converts specified DHCP options into the nodes tree.
     *
     * The resulting tree can comprise three different kinds of nodes:
     * - an option node, containing the basic option information (e.g., option code),
     * - option field node, containing field type and value,
     * - suboption node, i.e., a root of a suboption.
     *
     * It processes specified options recursively. It stops at recursion level of 3.
     *
     * @param options input options to be processed.
     * @param level option configuration level.
     * @param recursionLevel specifies current recursion level. It protects against
     *                       infinite recursion.
     * @returns parsed options as a displayable tree.
     */
    private convertOptionsToNodes(
        options: DHCPOption[],
        level: string,
        recursionLevel: number = 0
    ): TreeNode<OptionNode | OptionFieldNode>[] {
        let optionNodes: TreeNode<OptionNode | OptionFieldNode>[] = []
        if (!options || recursionLevel >= 3) {
            return optionNodes
        }
        for (let option of options) {
            // Parse option code and other parameters that don't belong to option payload.
            let optionNode: TreeNode<OptionNode | OptionFieldNode> = {
                type: recursionLevel === 0 ? 'option' : 'suboption',
                expanded: true,
                data: {
                    alwaysSend: option.alwaysSend,
                    code: option.code,
                    universe: option.universe,
                    level: level,
                },
                children: [],
            }
            // Iterate over the option fields.
            if (option.fields) {
                for (let field of option.fields) {
                    // Parse option field type and values.
                    let fieldNode: TreeNode<OptionNode | OptionFieldNode> = {
                        type: 'field',
                        expanded: true,
                        data: {
                            fieldType: field.fieldType,
                            value: field.values.join('/'),
                        },
                    }
                    optionNode.children.push(fieldNode)
                }
            }
            // Parse suboptions recursively.
            optionNode.children = optionNode.children.concat(
                this.convertOptionsToNodes(option.options, level, recursionLevel + 1)
            )
            optionNodes.push(optionNode)
        }
        return optionNodes
    }

    /**
     * Checks if the specified tree node represents an option with no suboptions.
     *
     * @param optionNode option node describing an option.
     * @returns true if the option comprises on suboptions.
     */
    isEmpty(optionNode: any): boolean {
        return (
            !optionNode.children ||
            optionNode.children.length === 0 ||
            optionNode.children.findIndex((c) => c.type === 'field') < 0
        )
    }

    /**
     * Returns a header displayed for an option.
     *
     * If the option is known, the returned string comprises option name
     * and option code. Otherwise, it comprises only the option code.
     *
     * @param code option code.
     * @returns option name followed by the option code or the string 'Option'
     *          followed by the option code.
     */
    getOptionTitle(node: TreeNode<OptionNode>): string {
        let option =
            node.data.universe === IPType.IPv4
                ? this.optionsService.findStandardDhcpv4Option(node.data.code)
                : this.optionsService.findStandardDhcpv6Option(node.data.code)
        if (option) {
            return `${option.label}`
        }
        return `Option ${node.data.code}`
    }

    /**
     * Returns severity of the tag displaying a configuration level for an option.
     * @param node a node containing an option descriptor.
     * @returns 'success' for the first configuration level, 'warning' for the second
     * configuration level, 'danger' for the third configuration level, and 'info' for
     * any other configuration level or when expected configuration levels do not exist.
     */
    getLevelTagSeverity(node: TreeNode<OptionNode>): string {
        if (this.levels?.length >= 1 && node.data?.level === this.levels[0]) {
            return 'success'
        }
        if (this.levels?.length >= 2 && node.data?.level === this.levels[1]) {
            return 'warning'
        }
        if (this.levels?.length >= 3 && node.data?.level === this.levels[2]) {
            return 'danger'
        }
        return 'info'
    }

    /**
     * An event handler invoked when clicking on a checkbox that toggles options
     * display mode.
     *
     * When the checkbox is on, only the first-level options are displayed. Otherwise,
     * all options are displayed including inheritance from all levels.
     */
    onCombinedChange() {
        this.displayedOptionNodes = this.currentLevelOnlyMode ? this.optionNodes[0] : this.combinedOptionNodes
    }
}
