import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { KeaDaemonConfig, ServicesService } from '../backend'
import { Subscription, lastValueFrom } from 'rxjs'
import { MessageService } from 'primeng/api'
import { daemonNameToFriendlyName, getErrorMessage } from '../utils'
import { ActivatedRoute } from '@angular/router'
import { NamedCascadedParameters } from '../cascaded-parameters-board/cascaded-parameters-board.component'
import { KeaGlobalConfigurationFormComponent } from '../kea-global-configuration-form/kea-global-configuration-form.component'
import { DHCPOption } from '../backend'

/**
 * A component that displays global configuration parameter for Kea.
 */
@Component({
    selector: 'app-kea-global-configuration-page',
    templateUrl: './kea-global-configuration-page.component.html',
    styleUrl: './kea-global-configuration-page.component.sass',
})
export class KeaGlobalConfigurationPageComponent implements OnInit, OnDestroy {
    @ViewChild(KeaGlobalConfigurationFormComponent) keaGlobalConfigurationForm: KeaGlobalConfigurationFormComponent

    /**
     * Breadcrumbs for this view.
     */
    breadcrumbs = [
        { label: 'Services' },
        { label: 'Kea Apps', routerLink: '/apps/kea/all' },
        { label: 'App' },
        { label: 'Daemons' },
        { label: 'Daemon' },
        { label: 'Global Configuration' },
    ]

    /**
     * Daemon ID for which the configuration is fetched.
     */
    daemonId: number

    /**
     * Daemon name fetched from the server.
     */
    daemonName: string

    /**
     * App ID to which the daemon belongs.
     */
    appId: number

    /**
     * App name for which the daemon belongs.
     */
    appName: string

    /**
     * Holds fetched configuration. It always contains one (or zero) element.
     */
    dhcpParameters: Array<NamedCascadedParameters<Record<string, any>>> = []

    /**
     * Holds fetched DHCP options. It always contains one (or zero) element.
     */
    dhcpOptions: DHCPOption[][] = []

    /**
     * Subscriptions released when the component is destroyed.
     */
    subscriptions = new Subscription()

    /**
     * A flag indicating when the data have been loaded from the server.
     */
    loaded: boolean = false

    /**
     * A boolean flag indicating if the configuration is currently edited.
     */
    edit: boolean = false

    /**
     * Boolean flag disabling the edit button in the configuration view.
     */
    disableEdit: boolean = true

    /**
     * A list of parameters not presented in this view but fetched from
     * the server in the configuration.
     */
    excludedParameters: Array<string> = [
        'clientClasses',
        'configControl',
        'hooksLibraries',
        'loggers',
        'optionData',
        'optionDef',
        'optionsHash',
        'reservations',
        'subnet4',
        'subnet6',
        'sharedNetworks',
    ]

    /**
     * Constructor.
     *
     * @param route an activated route used to fetch the current parameters.
     * @param messageService a message service used to display error messages.
     * @param servicesService a service used to communicate with the Kea servers.
     */
    constructor(
        private route: ActivatedRoute,
        private messageService: MessageService,
        private servicesService: ServicesService
    ) {}

    /**
     * Component lifecycle hook invoked during the component initialization.
     *
     * It fetches Kea configuration from the server.
     */
    ngOnInit(): void {
        this.subscriptions.add(
            this.route.paramMap.subscribe((params) => {
                const daemonIdStr = params.get('daemonId')
                const daemonId = parseInt(daemonIdStr, 10)
                this.daemonId = daemonId
                this.appId = parseInt(params.get('appId'), 10)
                this.updateBreadcrumbs(this.appId, this.daemonId)
                this.load()
            })
        )
    }

    /**
     * Component lifecycle hook invoked when the component is destroyed.
     *
     * It removes all subscriptions.
     */
    ngOnDestroy(): void {
        // When leaving the page the form can be still open. We have to
        // cancel the transaction if it hasn't been canceled.
        this.keaGlobalConfigurationForm?.onCancel()
        this.subscriptions.unsubscribe()
    }

    /**
     * Callback invoked when the user begins editing the configuration.
     */
    onEditBegin(): void {
        this.edit = true
    }

    /**
     * Callback invoked when the user clicks cancel in the form.
     */
    onFormCancel(): void {
        this.edit = false
    }

    /**
     * Callback invoked when the user submits updated configuration.
     */
    onFormSubmit(): void {
        this.edit = false
        // Reload the configuration after the update.
        this.load()
    }

    /**
     * Fetches Kea daemon configuration from the server.
     */
    private load(): void {
        this.loaded = false
        this.disableEdit = true
        lastValueFrom(this.servicesService.getDaemonConfig(this.daemonId))
            .then((data: KeaDaemonConfig) => {
                // Update daemon and app identifiers.
                this.daemonName = data.daemonName
                const friendlyDaemonName = daemonNameToFriendlyName(data.daemonName)
                this.appId = data.appId
                this.appName = data.appName

                // Update DHCP parameters.
                this.dhcpParameters = [
                    {
                        name: `${data.appName} / ${friendlyDaemonName}`,
                        parameters: [data.config.Dhcp4 ?? data.config.Dhcp6],
                    },
                ]
                this.disableEdit = !data.editable

                this.dhcpOptions = data.options ? [data.options.options] : []

                // Update breadcrumbs.
                this.updateBreadcrumbs(this.appId, this.daemonId, this.appName, friendlyDaemonName)
            })
            .catch((err) => {
                let msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Failed to fetch Kea daemon configuration',
                    detail: `Failed to fetch Kea daemon configuration: ${msg}`,
                    life: 10000,
                })
            })
            .finally(() => {
                this.loaded = true
            })
    }

    /** Updates the breadcrumbs links and labels. */
    updateBreadcrumbs(appId: number, daemonId: number, appName?: string, daemonName?: string): void {
        const breadcrumb = [...this.breadcrumbs]

        if (appId != null) {
            breadcrumb[2].routerLink = `/apps/kea/${appId}`
            if (daemonId != null) {
                breadcrumb[4].routerLink = `/apps/kea/${appId}?daemon=${daemonId}`
            }
        }

        if (appName != null) {
            breadcrumb[2].label = appName
        }

        if (daemonName != null) {
            breadcrumb[4].label = daemonName
        }

        this.breadcrumbs = breadcrumb
    }
}
