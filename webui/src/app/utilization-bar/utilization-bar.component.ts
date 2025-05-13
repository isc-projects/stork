import { Component, Input } from '@angular/core'
import { clamp } from '../utils'

@Component({
    selector: 'app-utilization-bar',
    templateUrl: './utilization-bar.component.html',
    styleUrl: './utilization-bar.component.sass',
})
export class UtilizationBarComponent {
    /**
     * Tooltip to display when hovering over the utilization bar.
     */
    @Input() tooltip: string | null = null

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
            width: clamp(Math.ceil(this.utilizationPrimary), 0, 100) + '%',
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
            width: clamp(Math.ceil(this.utilizationSecondary), 0, 100) + '%',
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
