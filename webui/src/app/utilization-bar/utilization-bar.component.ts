import { Component, Input } from '@angular/core'
import { clamp, datetimeToLocal, uncamelCase, unhyphen } from '../utils'

/**
 * A component displaying a utilization bar.
 *
 * It can be displayed as a single bar or as a double bar. The single bar
 * displays the utilization of a single kind (e.g. addresses, delegated
 * prefixes). The double bar displays the utilization of two kinds (e.g.
 * addresses and delegated prefixes) in a compact form. The bars are stacked
 * vertically and each of them occupy 50% of height. The both bars share the
 * same main label (e.g. subnet prefix) and each bar has its own kind label.
 *
 * The utilization bar may also display a tooltip with statistics on hover.
 */
@Component({
    selector: 'app-utilization-bar',
    standalone: false,
    templateUrl: './utilization-bar.component.html',
    styleUrl: './utilization-bar.component.sass',
})
export class UtilizationBarComponent {
    /**
     * Utilization value for the primary (top) bar.
     *
     * Value in percentages in range from 0 to 100.
     * It may be greater than 100[%] but such values are displayed with a
     * warning in the tooltip and special color on the bar.
     */
    @Input() utilizationPrimary: number | null = null
    /**
     * Kind label for the primary (top) bar.
     */
    @Input() kindPrimary: string | null = null

    /**
     * Utilization value for the secondary (bottom) bar.
     * This is used for displaying a double bar.
     * It may be null if the secondary bar is not used.
     *
     * Value in percentages in range from 0 to 100.
     * It may be greater than 100[%] but such values are displayed with a
     * warning in the tooltip and special color on the bar.
     */
    @Input() utilizationSecondary: number | null = null

    /**
     * Kind label for the secondary (bottom) bar.
     * It must be set to non-null if the secondary bar is used.
     */
    @Input() kindSecondary: string | null = null

    /**
     * Statistics related to the utilizations. Optional.
     * This is used for displaying a tooltip with more details.
     */
    @Input() stats: { [key: string]: number | bigint } | null = null

    /**
     * Statistics collection time. Optional.
     * This is used for displaying a tooltip with more details.
     */
    @Input() statsCollectedAt: string | null = null

    /**
     * Returns a content of the statistics tooltip.
     * It includes all the statistics, the collection time and the utilization
     * values.
     */
    get tooltip(): string {
        const lines: string[] = []

        if (this.utilizationPrimary > 100 || this.utilizationSecondary > 100) {
            lines.push('Warning! Utilization is greater than 100%. Data is unreliable.')
            lines.push(
                'This problem is caused by a Kea limitation - addresses/NAs/PDs in out-of-pool host reservations are reported as assigned but excluded from the total counters.'
            )
            lines.push(
                'Please manually check that the pool has free addresses and make sure that Kea and Stork are up-to-date.'
            )
            lines.push('')
        }

        let hasUtilizationLine = false
        if (this.utilizationPrimary != null) {
            let line = 'Utilization'
            if (this.kindPrimary && this.kindSecondary) {
                line += ` ${this.kindPrimary}`
            }
            line += `: ${this.utilizationPrimary.toFixed(1)}%`
            lines.push(line)
            hasUtilizationLine = true
        }

        if (this.isDouble && this.utilizationSecondary != null) {
            lines.push(`Utilization ${this.kindSecondary}: ${this.utilizationSecondary.toFixed(1)}%`)
            hasUtilizationLine = true
        }

        if (hasUtilizationLine) {
            lines.push('')
        }

        if (this.stats == null) {
            lines.push('No statistics yet')
        }

        // Sort the statistics by the second word in the key then by the first word.
        // It allows to group the statistics by the kind.
        // For example: "assigned-addresses" and "declined-prefixes" will be grouped together.
        const entries = Object.entries(this.stats ?? {}).sort(([keyA], [keyB]) => {
            const wordsA = keyA.split('-')
            const wordsB = keyB.split('-')
            if (wordsA.length <= 1 || wordsB.length <= 1) {
                // Single word statistics.
                return keyA.localeCompare(keyB)
            }

            const secondWordA = wordsA[1]
            const secondWordB = wordsB[1]
            if (secondWordA === secondWordB) {
                // If the second word is the same, sort by the whole statistic.
                return keyA.localeCompare(keyB)
            }
            // Otherwise, sort by the second word.
            return secondWordA.localeCompare(secondWordB)
        })

        for (const [key, value] of entries) {
            const formattedKey = uncamelCase(unhyphen(key))
            lines.push(`${formattedKey}: ${value.toLocaleString('en-US')}`)
        }

        if (this.statsCollectedAt) {
            lines.push('')
            lines.push(`Collected at: ${datetimeToLocal(this.statsCollectedAt) || 'never'}`)
        }

        return lines.join('<br>').trim()
    }

    /**
     * Indicates if the utilization bar is a double bar.
     */
    get isDouble(): boolean {
        return this.kindSecondary !== null
    }

    /**
     * Returns a style for the address utilization bar.
     */
    get utilizationStylePrimary() {
        return {
            // In some cases the utilization may be incorrect - less than
            // zero or greater than 100%. We need to truncate the value
            // to avoid a subnet bar overlapping other elements.
            width: clamp(Math.ceil(this.utilizationPrimary ?? 0), 0, 100) + '%',
        }
    }

    /**
     * Returns a style for the delegated prefix utilization bar.
     */
    get utilizationStyleSecondary() {
        return {
            // In some cases the utilization may be incorrect - less than
            // zero or greater than 100%. We need to truncate the value
            // to avoid a subnet bar overlapping other elements.
            width: clamp(Math.ceil(this.utilizationSecondary ?? 0), 0, 100) + '%',
        }
    }

    /**
     * Returns a proper CSS modificator class for a given utilization value.
     */
    getUtilizationBarModificatorClass(utilization: number | null): string {
        if (utilization == null) {
            return 'utilization__bar--missing'
        }
        if (utilization <= 80) {
            return 'utilization__bar--low'
        }
        if (utilization <= 90) {
            return 'utilization__bar--medium'
        }
        if (utilization <= 100) {
            return 'utilization__bar--high'
        }
        return 'utilization__bar--exceed'
    }
}
