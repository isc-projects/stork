import { Component, computed, EventEmitter, input, Output } from '@angular/core'

import { AnyDaemon, ServicesService } from '../backend'
import { daemonStatusIconClass, daemonStatusIconTooltip, getErrorMessage } from '../utils'
import { KeaDaemonComponent } from '../kea-daemon/kea-daemon.component'
import { Bind9DaemonComponent } from '../bind9-daemon/bind9-daemon.component'
import { PdnsDaemonComponent } from '../pdns-daemon/pdns-daemon.component'
import { Button } from 'primeng/button'
import { Tooltip } from 'primeng/tooltip'
import { isKeaDaemon } from '../version.service'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { NgIf } from '@angular/common'
import { ConfirmationService, MessageService } from 'primeng/api'
import { last } from 'rxjs'

@Component({
    selector: 'app-daemon-tab',
    templateUrl: './daemon-tab.component.html',
    styleUrl: './daemon-tab.component.sass',
    imports: [
        NgIf,
        Tooltip,
        Button,
        KeaDaemonComponent,
        Bind9DaemonComponent,
        PdnsDaemonComponent,
        EntityLinkComponent,
    ],
})
export class DaemonTabComponent {
    daemon = input<AnyDaemon>(null)
    /**
     * Indicates if the daemon is being deleted.
     */
    @Output() refreshDaemon = new EventEmitter<number>()
    @Output() deleteDaemon = new EventEmitter<number>()

    constructor(
        private servicesApi: ServicesService,
        private confirmService: ConfirmationService,
        private msgService: MessageService
    ) {}

    /**
     * The CSS class to display the icon to be used to indicate daemon status
     */
    daemonStatusIconClass = computed(() => daemonStatusIconClass(this.daemon()))

    /**
     * Tooltip for the icon presented for the daemon status
     */
    daemonStatusIconTooltip = computed(() => daemonStatusIconTooltip(this.daemon()))

    /**
     * Indicates if the given daemon is a Kea daemon.
     * @param daemon
     * @returns true if the daemon is Kea daemon; otherwise false.
     */
    isKeaDaemon = computed(() => isKeaDaemon(this.daemon()?.name))

    /**
     * Emits the refresh event.
     */
    refresh() {
        const daemon = this.daemon()
        if (daemon?.id !== undefined) {
            this.refreshDaemon.emit(daemon.id)
        }
    }

    /**
     * Displays a dialog to confirm daemon deletion.
     */
    confirmDelete() {
        this.confirmService.confirm({
            message:
                'Are you sure that you want to delete this daemon? <br/> Please note that the active daemon will be redetected automatically soon.',
            header: 'Delete Daemon',
            icon: 'pi pi-exclamation-triangle',
            rejectButtonProps: { text: true, icon: 'pi pi-times', label: 'Cancel' },
            acceptButtonProps: {
                icon: 'pi pi-check',
                label: 'Delete',
            },
            accept: () => {
                this.delete()
            },
        })
    }

    /**
     * Emits the delete event.
     */
    delete() {
        const daemon = this.daemon()
        this.servicesApi
            .deleteDaemon(daemon.id)
            .pipe(last())
            .subscribe({
                next: () => {
                    this.msgService.add({
                        severity: 'success',
                        summary: 'Daemon successfully deleted',
                    })
                    this.deleteDaemon.emit(daemon.id)
                },
                error: (err) => {
                    const msg = getErrorMessage(err)
                    this.msgService.add({
                        severity: 'error',
                        summary: 'Cannot delete the daemon',
                        detail: 'Failed to delete the daemon: ' + msg,
                        life: 10000,
                    })
                },
            })
    }
}
