import { Component, Input } from '@angular/core'
import { daemonNameToFriendlyName } from '../utils'

/**
 * A component that displays given entity as a link with rounded border
 * and a background color. It can be used for making a link to:
 * - machine
 * - app
 * - daemon
 * - subnet
 * - host
 */
@Component({
    selector: 'app-entity-link',
    standalone: false,
    templateUrl: './entity-link.component.html',
    styleUrls: ['./entity-link.component.sass'],
})
export class EntityLinkComponent {
    /**
     * Entity name, one of: machine, app, daemon, subnet, host.
     */
    @Input() entity: string

    /**
     * Attributes that describe given entity e.g. id, name, etc.
     */
    @Input() attrs: any

    /**
     * Boolean flag indicating if the entity name should be displayed.
     */
    @Input() showEntityName = true

    /**
     * Name of the class overriding original component style.
     */
    @Input() styleClass: string

    constructor() {}

    /**
     * Reference to daemonNameToFriendlyName() function to be used in html template.
     * @protected
     */
    protected readonly daemonNameToFriendlyName = daemonNameToFriendlyName
}
