import { Component, OnDestroy, OnInit } from '@angular/core'
import { ServicesService } from '../backend'
import { Subscription, lastValueFrom } from 'rxjs'
import { MessageService } from 'primeng/api'
import { getErrorMessage } from '../utils'
import { ActivatedRoute } from '@angular/router'
import { NamedCascadedParameters } from '../cascaded-parameters-board/cascaded-parameters-board.component'

@Component({
    selector: 'app-kea-global-configuration-page',
    templateUrl: './kea-global-configuration-page.component.html',
    styleUrl: './kea-global-configuration-page.component.sass',
})
export class KeaGlobalConfigurationPageComponent implements OnInit, OnDestroy {
    daemonId: number

    breadcrumbs = [{ label: 'DHCP' }, { label: 'Global Parameters' }, { label: 'Daemons' }, { label: 'Daemon' }]

    dhcpParameters: Array<NamedCascadedParameters<Object>> = []

    subscriptions = new Subscription()

    loaded: boolean = false

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
     */
    ngOnInit(): void {
        this.subscriptions.add(
            this.route.paramMap.subscribe((params) => {
                const daemonIdStr = params.get('daemonId')
                const daemonId = parseInt(daemonIdStr, 10)
                this.daemonId = daemonId

                this.loaded = false
                lastValueFrom(this.servicesService.getDaemonConfig(this.daemonId))
                    .then((data) => {
                        this.dhcpParameters.push({
                            name: 'Server 1',
                            parameters: [data.Dhcp4],
                        })
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
            })
        )
    }

    /**
     * Component lifecycle hook invoked when the component is destroyed.
     *
     * It removes all subscriptions.
     */
    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }
}
