import { Component, EventEmitter, Input, Output } from '@angular/core'
import { ZoneRRs } from '../backend/model/zoneRRs'
import { DNSService } from '../backend/api/dNS.service'
import { lastValueFrom } from 'rxjs'
import { MessageService } from 'primeng/api'
import { getErrorMessage } from '../utils'

/**
 * A component responsible for fetching the zone resource records and
 * passing them to the zone viewer for display.
 */
@Component({
    selector: 'app-zone-viewer-feeder',
    standalone: false,
    templateUrl: './zone-viewer-feeder.component.html',
    styleUrl: './zone-viewer-feeder.component.sass',
})
export class ZoneViewerFeederComponent {
    /**
     * Holds the daemon ID.
     */
    @Input({ required: true }) daemonId: number

    /**
     * Holds the DNS view name.
     */
    @Input({ required: true }) viewName: string

    /**
     * Holds the zone ID.
     */
    @Input({ required: true }) zoneId: number

    /**
     * Sets the flag indicating that the component should fetch the
     * zone resource records.
     */
    @Input({ required: true }) set active(active: boolean) {
        if (active && !this._loaded) {
            this._loadRRs()
        }
    }

    /**
     * Emits the event indicating that fetching the zone resource records
     * has failed.
     *
     * The parent component can use this event to take specific actions
     * like hiding the zone viewer dialog.
     */
    @Output() viewerError = new EventEmitter<void>()

    /**
     * Holds the zone resource records.
     */
    zoneData: ZoneRRs = {
        items: [],
    }

    /**
     * Holds the timestamp of the last zone transfer.
     */
    zoneTransferAt: string = null

    /**
     * Holds the flag indicating that the zone resource records have been loaded.
     *
     * It is used to prevent loading the zone resource records multiple times.
     */
    _loaded: boolean = false

    /**
     * Holds the flag indicating that the zone resource records are being loaded.
     *
     * It is used to display the loading spinner.
     */
    loading: boolean = false

    /**
     * Constructor.
     *
     * @param _dnsApi DNS API service.
     * @param _messageService message service.
     */
    constructor(
        private _dnsApi: DNSService,
        private _messageService: MessageService
    ) {}

    /**
     * Loads the zone resource records from the server.
     */
    private _loadRRs(): void {
        // Show the loading spinner.
        this.loading = true
        lastValueFrom(this._dnsApi.getZoneRRs(this.daemonId, this.viewName, this.zoneId))
            .then((data) => {
                // The data have been successfully loaded.
                this._loaded = true
                this.zoneData = { items: data.items }
                this.zoneTransferAt = data.zoneTransferAt
            })
            .catch((error) => {
                // Show the error message.
                const errorMsg = getErrorMessage(error)
                this._messageService.add({
                    severity: 'error',
                    summary: 'Error getting zone contents',
                    detail: errorMsg,
                    life: 10000,
                })
                // Notify the parent.
                this.viewerError.emit()
            })
            .finally(() => {
                // Hide the loading spinner, regardless of the result.
                this.loading = false
            })
    }

    /**
     * Refreshes the zone contents from the DNS server.
     */
    private _refreshRRs(): void {
        // Show the loading spinner.
        this.loading = true
        lastValueFrom(this._dnsApi.putZoneRRsCache(this.daemonId, this.viewName, this.zoneId))
            .then((data) => {
                // The data have been successfully loaded.
                this._loaded = true
                this.zoneData = { items: data.items }
                this.zoneTransferAt = data.zoneTransferAt
            })
            .catch((error) => {
                // Show the error message.
                const errorMsg = getErrorMessage(error)
                this._messageService.add({
                    severity: 'error',
                    summary: 'Error refreshing zone contents from DNS server',
                    detail: errorMsg,
                    life: 10000,
                })
                // Notify the parent.
                this.viewerError.emit()
            })
            .finally(() => {
                // Hide the loading spinner, regardless of the result.
                this.loading = false
            })
    }

    /**
     * Refreshes the zone contents from the DNS server.
     */
    public refreshFromDNS() {
        this._refreshRRs()
    }
}
