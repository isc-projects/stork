import { Component, Input } from '@angular/core'
import { ZoneRRs } from '../backend/model/zoneRRs'
import { ZoneRR } from '../backend/model/zoneRR'

/**
 * Component that displays zone contents (resource records) in a table.
 *
 * It compacts presented data by removing the zone name (gathered from the SOA record)
 * from the resource records. It also omits the name from the resource record when
 * the previous resource record has the same name.
 */
@Component({
    selector: 'app-zone-viewer',
    templateUrl: './zone-viewer.component.html',
    styleUrl: './zone-viewer.component.sass',
})
export class ZoneViewerComponent {
    /**
     * Holds presented zone resource records.
     */
    private _data: ZoneRRs = {
        items: [],
    }

    /**
     * Holds the name of the zone gathered from the SOA record.
     */
    private _zoneName: string | null = null

    /**
     * Holds the name of the last processed resource record.
     *
     * It is used to omit repeated names in subsequent resource records.
     */
    private _lastName: string | null = null

    /**
     * Sets the zone resource records to be presented.
     *
     * This function compacts the received information by removing the zone name
     * from the resource record names and omitting the name from the resource record
     * when the previous resource record has the same name.
     *
     * @param rrs zone resource records before transformation.
     */
    @Input({ required: true })
    set data(rrs: ZoneRRs) {
        // Reset the internal state.
        this._zoneName = null
        this._lastName = null
        if (rrs?.items) {
            // Compact and assign the resource records.
            this._data.items = rrs.items.map((rr, index) => this._transformZoneRR(rr, index === 0)).filter((rr) => rr)
        } else {
            this._data.items = []
        }
    }

    /**
     * Returns the transformed resource records.
     */
    get data(): ZoneRRs {
        return this._data
    }

    /**
     * Transforms the resource record.
     *
     * @param rr resource record to transform.
     * @param isFirst flag indicating if the resource record is the first one.
     * @returns transformed resource record or null if the resource record should be omitted.
     */
    private _transformZoneRR(rr: ZoneRR, isFirst: boolean): ZoneRR | null {
        let name: string = ''
        switch (true) {
            case rr.rrType === 'SOA':
                if (!isFirst) {
                    // Zone transfer returns a duplicate SOA record at the end.
                    // It can be used for data integrity verification. However, we
                    // don't need to display it in the zone viewer. We want the viewer
                    // to display the data in the similar manner as in the zone file.
                    // The zone file lacks the duplicate SOA record.
                    return null
                }
                // The 'at' symbol is used in the zone file to represent the zone name.
                name = '@'
                this._zoneName = rr.name
                break
            case this._lastName === rr.name:
                // If subsequent resource record has the same name as the previous one,
                // let's omit the name (leave it empty).
                break
            case !!this._zoneName:
                // If we have the zone name (which we should after processing the SOA record),
                // we can remove it from the current resource record name.
                name = rr.name.replace(`.${this._zoneName}`, '')
                break
            default:
                // In all other cases let's just use what we have.
                name = rr.name
        }
        // Remember the current name as the last processed name.
        this._lastName = rr.name

        // Return the transformed resource record.
        return {
            ...rr,
            name,
        }
    }
}
