import { Component, Input } from '@angular/core'
import { clamp, datetimeToLocal, uncamelCase, unhyphen } from '../utils'

@Component({
    selector: 'app-utilization-bar',
    templateUrl: './utilization-bar.component.html',
    styleUrl: './utilization-bar.component.sass',
})
export class UtilizationBarComponent {
    /**
     * Utilization value for the primary (top) bar.
     */
    @Input() utilizationPrimary: number | null = null
    /**
     * Kind label for the primary (top) bar.
     */
    @Input() kindPrimary: string | null = null

    /**
     * Utilization value for the secondary (bottom) bar.
     * This is used for displaying a double bar.
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

    get tooltip(): string {
        if (this.stats == null) {
            return 'No statistics yet'
        }

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
          line += `: ${this.utilizationPrimary}%`
          lines.push(line)
          hasUtilizationLine = true
        }

        if (this.isDouble && this.utilizationSecondary != null) {
            lines.push(`Utilization ${this.kindSecondary}: ${this.utilizationSecondary}%`)
            hasUtilizationLine = true
        }

        if (hasUtilizationLine) {
          lines.push('')
        }

        // Sort the statistics by the second word in the key then by the first word.
        // It allows to group the statistics by the kind.
        // For example: "assigned-addresses" and "declined-prefixes" will be grouped together.
        const entries = Object.entries(this.stats).sort(([keyA], [keyB]) => {
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

        lines.push('')
        lines.push(`Collected at: ${datetimeToLocal(this.statsCollectedAt) || 'never'}`)

        return lines.join('<br>')
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
