import { Component, Input, OnInit } from '@angular/core'
import { App, Bind9Daemon, KeaDaemon } from '../backend'
import { TreeNode } from 'primeng/api'
import { daemonNameToFriendlyName } from '../utils'

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
 * A component displaying communication issues with the monitored apps
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
    standalone: false,
    templateUrl: './communication-status-tree.component.html',
    styleUrl: './communication-status-tree.component.sass',
})
export class CommunicationStatusTreeComponent implements OnInit {
    /**
     * A list of apps having communication issues returned by the server.
     */
    @Input() apps: App[] = []

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
        // Go over the returned apps.
        for (const app of this.apps) {
            // Each app should contain a reference to the machine where it belongs.
            // Let's see if the top level nodes already contain this machine. It is
            // possible that multiple apps are running on the same machine, so it
            // could have been already added.
            let machineNode: TreeNode<CommunicationStatusNodeData> = this.nodes.find(
                (node) => node.data?.attrs?.id === app.machine?.id
            )
            // If this is the first time we see this machine, let's add it.
            if (!machineNode) {
                machineNode = {
                    icon: 'pi pi-server',
                    type: 'machine',
                    children: [],
                    styleClass: this.getStyleClassForErrors(true, 0),
                    expanded: true,
                    data: {
                        attrs: {
                            id: app.machine?.id,
                            address: app.machine?.address,
                        },
                    },
                }
                this.nodes.push(machineNode)
            }
            // Processing will be different depending on the app type.
            switch (app.type) {
                case 'kea':
                    // Kea apps typically have a Kea Control Agent between the Stork
                    // Agent and the daemons. Let's try to find the Kea Control Agent
                    // among the returned daemons.
                    let currentNode = machineNode
                    const daemons = app.details?.daemons || []
                    const caDaemon = daemons.find((daemon) => this.isKeaControlAgent(daemon))
                    if (caDaemon) {
                        // Found the Kea Control Agent. Let's create a new node representing
                        // the Kea Control Agent and add it below the Stork Agent node.
                        currentNode = {
                            icon: 'pi pi-sitemap',
                            type: app.type,
                            expanded: true,
                            children: [],
                            styleClass: this.getStyleClassForErrors(true, caDaemon.caCommErrors),
                            data: {
                                attrs: {
                                    id: app.id,
                                    type: app.type,
                                    name: app.name,
                                },
                                caCommErrors: caDaemon.caCommErrors,
                            },
                        }
                        machineNode.children.push(currentNode)
                    }
                    // Now, let's iterate over the rest of the daemons.
                    for (let daemon of daemons) {
                        // The daemon may contain the number of the communication errors
                        // with the Stork Agent. Let's record it.
                        this.updateMachineCommErrors(daemon, machineNode)
                        // We have already processed the Kea CA.
                        if (this.isKeaControlAgent(daemon)) {
                            continue
                        }
                        // Add a node representing a Kea daemon.
                        currentNode.children.push({
                            icon: 'pi pi-link',
                            styleClass: this.getStyleClassForErrors(daemon.monitored, daemon.daemonCommErrors),
                            type: 'kea-daemon',
                            expanded: true,
                            data: {
                                attrs: {
                                    id: daemon.id,
                                    appType: app.type,
                                    appId: app.id,
                                    name: daemonNameToFriendlyName(daemon.name),
                                },
                                daemonCommErrors: daemon.daemonCommErrors,
                                monitored: !!daemon.monitored,
                            },
                        })
                    }
                    break

                case 'bind9':
                    // BIND 9 daemon is held in a different structure.
                    const daemon = app.details?.daemon
                    if (!daemon) {
                        continue
                    }
                    // It may hold the number of errors to communicate with the Stork Agent.
                    // Let's record it.
                    this.updateMachineCommErrors(daemon, machineNode)
                    // Let's create two subnodes representing the control and stats channels.
                    machineNode.children.push(
                        this.createNamedChannelNode(app, daemon, 'rndc'),
                        this.createNamedChannelNode(app, daemon, 'stats')
                    )
                    break

                default:
                    // Other apps are not supported at this point.
                    break
            }
        }
    }

    /**
     * Convenience function checking if a daemon is a Kea Control Agent.
     *
     * @param daemon Kea daemon instance.
     * @returns true if the daemon is a Kea Control Agent, false otherwise.
     */
    private isKeaControlAgent(daemon: KeaDaemon): boolean {
        return daemon && (!daemon.name || daemon.name === 'ca')
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
        app: App,
        daemon: Bind9Daemon,
        channelType: 'rndc' | 'stats'
    ): TreeNode<CommunicationStatusNodeData> {
        return {
            icon: `pi pi-link`,
            type: 'named-channel',
            styleClass: this.getStyleClassForErrors(
                !!daemon.monitored,
                channelType === 'rndc' ? daemon.rndcCommErrors : daemon.statsCommErrors
            ),
            data: {
                channelName: channelType === 'rndc' ? 'Control' : 'Statistics',
                attrs: {
                    id: daemon.id,
                    appType: app.type,
                    appId: app.id,
                    name: daemonNameToFriendlyName(daemon.name),
                },
                channelCommErrors: channelType === 'rndc' ? daemon.rndcCommErrors : daemon.statsCommErrors,
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
        daemon: KeaDaemon | Bind9Daemon,
        machineNode: TreeNode<CommunicationStatusNodeData>
    ): void {
        if (daemon?.agentCommErrors > 0) {
            machineNode.styleClass = this.getStyleClassForErrors(true, daemon?.agentCommErrors)
            machineNode.data.agentCommErrors = daemon.agentCommErrors
        }
    }
}
