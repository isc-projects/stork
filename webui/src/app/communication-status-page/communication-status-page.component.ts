import { Component, OnInit } from '@angular/core'
import { AnyDaemon, ServicesService } from '../backend'
import { lastValueFrom } from 'rxjs'
import { getErrorMessage } from '../utils'
import { MessageService } from 'primeng/api'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { NgIf } from '@angular/common'
import { CommunicationStatusTreeComponent } from '../communication-status-tree/communication-status-tree.component'
import { Button } from 'primeng/button'
import { ProgressSpinner } from 'primeng/progressspinner'

/**
 * A component displaying a page showing the communication issues with
 * the monitored daemons.
 *
 * It fetches the list of communication issues from the Stork server and
 * displays them as a tree using the CommunicationStatusTreeComponent.
 *
 * The list can be reloaded on demand by pressing a button.
 */
@Component({
    selector: 'app-communication-status-page',
    templateUrl: './communication-status-page.component.html',
    styleUrl: './communication-status-page.component.sass',
    imports: [BreadcrumbsComponent, NgIf, CommunicationStatusTreeComponent, Button, ProgressSpinner],
})
export class CommunicationStatusPageComponent implements OnInit {
    /**
     * Configures the breadcrumbs for the component.
     */
    breadcrumbs = [{ label: 'Monitoring' }, { label: 'Communication' }]

    /**
     * A list of communication issues fetched from the server.
     */
    daemons: Array<AnyDaemon> = []

    /**
     * A boolean flag indicating if the data are being loaded.
     */
    loading = true

    /**
     * Constructor.
     *
     * @param messageService message service used for displaying communication
     *        errros with the Stork server.
     * @param servicesService a service used for fetching the communication issues
     *        from the Stork server.
     */
    constructor(
        private messageService: MessageService,
        private servicesService: ServicesService
    ) {}

    /**
     * A component lifecycle hook invoked when the component is loaded.
     *
     * It fetches the list of communication issues from the server.
     */
    ngOnInit(): void {
        this.reload()
    }

    /**
     * Reloads the list of the communication issues from the server.
     *
     * It is called during the component initialization and on demand, when
     * the reload button is pressed.
     */
    private reload(): void {
        this.loading = true
        lastValueFrom(this.servicesService.getDaemonsWithCommunicationIssues())
            .then((data) => {
                this.daemons = data.items ?? []
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.messageService.add({
                    severity: 'error',
                    summary: 'Cannot create new transaction',
                    detail: 'Failed to create transaction for adding new host: ' + msg,
                    life: 10000,
                })
                this.daemons = []
            })
            .finally(() => {
                this.loading = false
            })
    }

    /**
     * Reloads the list of communication issues on demand.
     *
     * It contacts the Stork server to fetch the list of communication issues.
     */
    onReload(): void {
        this.reload()
    }
}
