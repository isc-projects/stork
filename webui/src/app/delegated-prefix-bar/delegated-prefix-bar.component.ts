import { Component, Input } from '@angular/core'
import { DelegatedPrefixPool } from '../backend'
import { formatShortExcludedPrefix } from '../utils'
import { UtilizationBarComponent } from '../utilization-bar/utilization-bar.component'
import { NgIf } from '@angular/common'

/**
 * Displays the delegated prefix in a bar form.
 * Supports the delegated prefix with an excluded part.
 * See: [RFC 6603](https://www.rfc-editor.org/rfc/rfc6603.html).
 */
@Component({
    selector: 'app-delegated-prefix-bar',
    templateUrl: './delegated-prefix-bar.component.html',
    styleUrls: ['./delegated-prefix-bar.component.sass'],
    imports: [UtilizationBarComponent, NgIf],
})
export class DelegatedPrefixBarComponent {
    /**
     * The delegated prefix object. It may contain the excluded prefix.
     */
    @Input() pool: DelegatedPrefixPool

    /**
     * Returns the short representation of the excluded prefix.
     */
    get shortExcludedPrefix(): string {
        try {
            return formatShortExcludedPrefix(this.pool.prefix, this.pool.excludedPrefix)
        } catch {
            // Invalid prefix.
            return this.pool.excludedPrefix
        }
    }
}
