import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core'
import { KeaDaemonConfig, ServicesService } from '../backend'
import { Subscription, lastValueFrom } from 'rxjs'
import { MenuItem, MessageService } from 'primeng/api'
import { daemonNameToFriendlyName, getErrorMessage } from '../utils'
import { ActivatedRoute, RouterLink } from '@angular/router'
import { NamedCascadedParameters } from '../cascaded-parameters-board/cascaded-parameters-board.component'
import { KeaGlobalConfigurationFormComponent } from '../kea-global-configuration-form/kea-global-configuration-form.component'
import { DHCPOption } from '../backend'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { NgIf } from '@angular/common'
import { KeaGlobalConfigurationViewComponent } from '../kea-global-configuration-view/kea-global-configuration-view.component'
import { ProgressSpinner } from 'primeng/progressspinner'
import { DaemonNiceNamePipe } from '../pipes/daemon-name.pipe'

/**
 * A component that displays global configuration parameter for Kea.
 */
@Component({
    selector: 'app-kea-global-configuration-page',
    templateUrl: './kea-global-configuration-page.component.html',
    styleUrl: './kea-global-configuration-page.component.sass',
    imports: [
        BreadcrumbsComponent,
        NgIf,
        RouterLink,
        KeaGlobalConfigurationFormComponent,
        KeaGlobalConfigurationViewComponent,
        ProgressSpinner,
        DaemonNiceNamePipe,
    ],
})
export class KeaGlobalConfigurationPageComponent implements OnInit, OnDestroy {
    @ViewChild(KeaGlobalConfigurationFormComponent) keaGlobalConfigurationForm: KeaGlobalConfigurationFormComponent

    /**
     * Breadcrumbs for this view.
     */
    breadcrumbs: MenuItem[] = [
        { label: 'Services' },
        { label: 'Daemons', routerLink: '/daemons/all' },
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
                this.updateBreadcrumbs(this.daemonId)
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

                // Update DHCP parameters.
                this.dhcpParameters = [
                    {
                        name: friendlyDaemonName,
                        parameters: [data.config.Dhcp4 ?? data.config.Dhcp6],
                    },
                ]
                this.disableEdit = !data.editable

                this.dhcpOptions = data.options ? [data.options.options] : []

                // Update breadcrumbs.
                this.updateBreadcrumbs(this.daemonId, data.daemonName)
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
    updateBreadcrumbs(daemonId: number, daemonName?: string): void {
        const breadcrumb = [...this.breadcrumbs]

        if (daemonId != null) {
            breadcrumb[2].routerLink = `/daemons/${daemonId}`
        }

        if (daemonName != null) {
            breadcrumb[2].label = daemonNameToFriendlyName(daemonName)
        }

        this.breadcrumbs = breadcrumb
    }
}
