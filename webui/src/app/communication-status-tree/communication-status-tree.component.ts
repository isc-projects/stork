import { Component, Input, OnInit } from '@angular/core'
import { AnyDaemon, Bind9Daemon } from '../backend'
import { PrimeTemplate, TreeNode } from 'primeng/api'
import { Tree } from 'primeng/tree'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { NgIf } from '@angular/common'
import { Tooltip } from 'primeng/tooltip'

/**
 * Metadata associated with a tree node presenting communication
 * issues with an agent or daemon.
 */
interface CommunicationStatusNodeData {
    /**
     * Attributes passed to an entity link (e.g., machine, app).
     */
    attrs?: any
    /**
     * BIND 9 communication channel name to which the tree node pertains.
     */
    channelName?: 'Control' | 'Statistics'
    /**
     * Indicates if a daemon is monitored.
     */
    monitored?: boolean
    /**
     * Number of communication errors with a Stork agent.
     */
    agentCommErrors?: number
    /**
     * Number of communication errors with a Kea Control Agent.
     */
    caCommErrors?: number
    /**
     * Number of communication errors with a Kea daemon.
     */
    daemonCommErrors?: number
    /**
     * Number of communication errors via a BIND 9 channel.
     */
    channelCommErrors?: number
}

/**
 * A component displaying communication issues with the monitored daemons
 * as a tree.
 *
 * The top nodes of the tree represent Stork agents running on the monitored
 * machines. Next level for Kea holds a state of the Kea Control Agent. Finally,
 * the lowest level contains a list of Kea daemons behind the Kea Control Agent.
 *
 * BIND 9-specific tree has a slightly different structure. The level below the
 * Stork agent contains a list of communication channels for the named daemon
 * (i.e., control channel and statistics channel).
 *
 * The failing nodes are colored and an exclamation mark is shown next to them.
 */
@Component({
    selector: 'app-communication-status-tree',
    templateUrl: './communication-status-tree.component.html',
    styleUrl: './communication-status-tree.component.sass',
    imports: [Tree, PrimeTemplate, EntityLinkComponent, NgIf, Tooltip],
})
export class CommunicationStatusTreeComponent implements OnInit {
    /**
     * A list of apps having communication issues returned by the server.
     */
    @Input() daemons: AnyDaemon[] = []

    /**
     * A tree reflecting the hierarchy of the agents and daemons holding
     * the necessary metadata to display the messages about failures.
     */
    nodes: Array<TreeNode<CommunicationStatusNodeData>> = []

    /**
     * A component lifecycle hook called during the component initialization.
     *
     * It processes the input list of apps and creates a tree of the failing
     * nodes, along with the necessary metadata.
     */
    ngOnInit(): void {
        for (const daemon of this.daemons) {
            const machineId = daemon.machineId
            let machineNode = this.nodes.find((node) => node.data?.attrs?.machineId === machineId)
            if (!machineNode) {
                machineNode = {
                    icon: 'pi pi-server',
                    type: 'machine',
                    children: [],
                    styleClass: this.getStyleClassForErrors(true, daemon.agentCommErrors ?? 0),
                    expanded: true,
                    data: {
                        attrs: daemon,
                    },
                }
                this.nodes.push(machineNode)
            }

            // Bind9 daemons expose channel-level errors.
            if (daemon.name === 'named') {
                this.updateMachineCommErrors(daemon, machineNode)
                machineNode.children.push(
                    this.createNamedChannelNode(daemon as Bind9Daemon, 'rndc'),
                    this.createNamedChannelNode(daemon as Bind9Daemon, 'stats')
                )
                continue
            }

            // Kea-related daemons (legacy appType="kea") are the following names.
            const keaNames = new Set(['dhcp4', 'dhcp6', 'ca', 'd2', 'netconf'])
            const isKeaDaemon = keaNames.has(daemon.name)

            this.updateMachineCommErrors(daemon, machineNode)

            machineNode.children.push({
                icon: isKeaDaemon ? 'pi pi-sitemap' : 'pi pi-link',
                styleClass: this.getStyleClassForErrors(
                    daemon.monitored,
                    (daemon.daemonCommErrors ?? 0) + (daemon.caCommErrors ?? 0)
                ),
                type: isKeaDaemon ? 'kea' : 'other',
                expanded: true,
                data: {
                    attrs: daemon,
                    daemonCommErrors: daemon.daemonCommErrors,
                    caCommErrors: daemon.caCommErrors,
                    monitored: !!daemon.monitored,
                },
            })
        }
    }

    /**
     * Returns style class for a node based on the error count.
     *
     * @param monitored a boolean value indicating if the daemon or app is monitored.
     * @param errorCount communication error count for a node.
     * @returns Style class name highlighting a communication failure when
     * the error count is greater than 0, a style class name indicating
     * no communication failure or a style class indicating that the entity
     * is not monitored.
     */
    private getStyleClassForErrors(monitored?: boolean, errorCount?: number) {
        if (!monitored) {
            return 'communication-disabled'
        } else if (errorCount > 0) {
            return 'communication-failing'
        }
        return 'communication-ok'
    }

    /**
     * Instantiates a tree node for BIND 9 control or statistics channel.
     *
     * @param app BIND 9 app instance.
     * @param daemon BIND 9 daemon.
     * @param channelType channel type for which the node should be created.
     * @returns An instance of the tree node.
     */
    private createNamedChannelNode(
        daemon: Bind9Daemon,
        channelType: 'rndc' | 'stats'
    ): TreeNode<CommunicationStatusNodeData> {
        return {
            icon: `pi pi-link`,
            type: 'named-channel',
            styleClass: this.getStyleClassForErrors(
                !!daemon.monitored,
                channelType === 'rndc' ? daemon.daemonCommErrors : daemon.statsCommErrors
            ),
            data: {
                channelName: channelType === 'rndc' ? 'Control' : 'Statistics',
                attrs: daemon,
                channelCommErrors: channelType === 'rndc' ? daemon.daemonCommErrors : daemon.statsCommErrors,
                monitored: !!daemon.monitored,
            },
        }
    }

    /**
     * Updates a specified node based on the number of the communication
     * errors in daemon.
     *
     * The function sets the style of the node and the number of the communication
     * errors when this number is greater than 0.
     *
     * @param daemon a daemon instance holding the number of communication errors
     *        with the Stork Agent.
     * @param machineNode a tree node representing Stork Agent to be updated.
     */
    private updateMachineCommErrors(
        daemon: AnyDaemon | Bind9Daemon,
        machineNode: TreeNode<CommunicationStatusNodeData>
    ): void {
        if (daemon?.agentCommErrors > 0) {
            machineNode.styleClass = this.getStyleClassForErrors(true, daemon?.agentCommErrors)
            machineNode.data.agentCommErrors = daemon.agentCommErrors
        }
    }
}
