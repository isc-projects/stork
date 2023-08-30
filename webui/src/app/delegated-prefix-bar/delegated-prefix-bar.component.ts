import { Component, Input } from '@angular/core'
import { DelegatedPrefixPool } from '../backend'
import { formatShortExcludedPrefix } from '../utils'

/**
 * Displays the delegated prefix in a bar form.
 * Supports the delegated prefix with an excluded part.
 * See: [RFC 6603](https://www.rfc-editor.org/rfc/rfc6603.html).
 */
@Component({
    selector: 'app-delegated-prefix-bar',
    templateUrl: './delegated-prefix-bar.component.html',
    styleUrls: ['./delegated-prefix-bar.component.sass'],
})
export class DelegatedPrefixBarComponent {
    /**
     * The delegated prefix object. It may contain the excluded prefix.
     */
    @Input() prefix: DelegatedPrefixPool

    /**
     * Returns the short representation of the excluded prefix.
     */
    get shortExcludedPrefix(): string {
        try {
            return formatShortExcludedPrefix(this.prefix.prefix, this.prefix.excludedPrefix)
        } catch {
            // Invalid prefix.
            return this.prefix.excludedPrefix
        }
    }
}
