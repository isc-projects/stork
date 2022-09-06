import { Component, Input, OnInit } from '@angular/core'
import { TreeNode } from 'primeng/api'
import { DHCPOption } from '../backend/model/dHCPOption'
import { DhcpOptionsService } from '../dhcp-options.service'
import { IPType } from '../iptype'

/**
 * A node of the displayed option holding its basic description.
 *
 * It currently holds the option code and whether or not the option
 * is always returned by the DHCP server.
 */
export interface OptionNode {
    alwaysSend?: boolean
    code: number
    universe: IPType
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
     */
    @Input() options: Array<DHCPOption>

    /**
     * Collection of the converted options into the nodes that can be
     * displayed as a tree.
     */
    optionNodes: TreeNode<OptionNode | OptionFieldNode>[] = []

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
        this.optionNodes = this._convertOptionsToNodes(this.options)
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
     * @param recursionLevel specifies current recursion level. It protects against
     *                       infinite recursion.
     * @returns parsed options as a displayable tree.
     */
    private _convertOptionsToNodes(
        options: DHCPOption[],
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
                this._convertOptionsToNodes(option.options, recursionLevel + 1)
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
}
