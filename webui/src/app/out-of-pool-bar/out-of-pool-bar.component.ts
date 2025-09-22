import { Component, Input } from '@angular/core'

/**
 * A component displaying out of pool utilization bar. It selects and
 * normalizes statistic names related to out of pool usage.
 */
@Component({
    selector: 'app-out-of-pool-bar',
    standalone: false,
    templateUrl: './out-of-pool-bar.component.html',
    styleUrl: './out-of-pool-bar.component.sass',
})
export class OutOfPoolBarComponent {
    /**
     * Out of pool statistics. It contains only entries that are not
     * associated with any address pool. The keys are the names of the
     * statistics without the "out-of-pool" infix.
     */
    private _stats: { [key: string]: number | bigint | string } | null = null

    /**
     * Subnet or shared network statistics. They are used to display the out
     * of pool utilization.
     * If the statistics are not set or empty after filtering, the bar will not
     * be displayed.
     */
    @Input() set stats(value: { [key: string]: number | bigint | string } | null) {
        if (value == null) {
            this._stats = null
            return
        }

        // Filter out the entries that are related to out-of-pool usage.
        this._stats = Object.fromEntries(
            Object.entries(value)
                .filter(([key]) => key.includes('out-of-pool'))
                .filter(([key]) => (this.isPD ? key.endsWith('pds') : key.endsWith('addresses') || key.endsWith('nas')))
                .map(([key, val]) => [key.replace('out-of-pool-', ''), val])
        )
    }

    /**
     * Statistics collection time. Optional.
     * This is used for displaying a tooltip with more details.
     */
    @Input() statsCollectedAt: string | null = null

    /**
     * Utilization value for the out of pool bar.
     * Value in percentages in range from 0 to 100.
     * If the value is null or undefined, the bar will not be displayed.
     */
    @Input() utilization: number | null

    /**
     * Indicates whether the delegated prefix statistics should be extracted
     * from all provided statistics.
     * If true, the statistics will be filtered to include only those related
     * to delegated prefixes.
     * If false, the statistics will be filtered to include only those related
     * to addresses.
     */
    @Input() isPD: boolean = false

    /**
     * Returns the out of pool statistics. The names of the statistics
     * are without the "out-of-pool" infix.
     * @returns The out of pool statistics or null if not set.
     */
    get stats(): { [key: string]: number | bigint | string } | null {
        return this._stats
    }

    /**
     * Indicates whether the out of pool bar should be displayed.
     */
    get hasOutOfPoolData(): boolean {
        return (
            // The statistics must be set.
            this.stats != null &&
            // Any total statistics must be non-zero.
            // Warning: the counters must be compared with non-strict equality
            // because they may be BigInt or number or string.
            (('total-nas' in this.stats && this.stats['total-nas'] != 0) ||
                ('total-addresses' in this.stats && this.stats['total-addresses'] != 0) ||
                ('total-pds' in this.stats && this.stats['total-pds'] != 0))
        )
    }
}
