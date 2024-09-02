import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { KeaDaemonConfig, ServicesService } from '../backend'
import { Subscription, lastValueFrom } from 'rxjs'
import { MessageService } from 'primeng/api'
import { daemonNameToFriendlyName, getErrorMessage } from '../utils'
import { ActivatedRoute } from '@angular/router'
import { NamedCascadedParameters } from '../cascaded-parameters-board/cascaded-parameters-board.component'
import { KeaGlobalConfigurationFormComponent } from '../kea-global-configuration-form/kea-global-configuration-form.component'

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
     * Holds fetched configuration.
     */
    dhcpParameters: Array<NamedCascadedParameters<Object>> = []

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
                this.daemonName = data.daemonName
                this.appId = data.appId
                this.appName = data.appName
                this.dhcpParameters = [
                    {
                        name: `${data.appName} / ${daemonNameToFriendlyName(data.daemonName)}`,
                        parameters: [data.config.Dhcp4 ?? data.config.Dhcp6],
                    },
                ]
                this.disableEdit = !data.editable
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
}
