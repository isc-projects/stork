import { Component, Input } from '@angular/core'
import { NgSwitch, NgClass, NgSwitchCase, NgSwitchDefault } from '@angular/common'
import { RouterLink } from '@angular/router'
import { DaemonNiceNamePipe } from '../pipes/daemon-name.pipe'

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
    templateUrl: './entity-link.component.html',
    styleUrls: ['./entity-link.component.sass'],
    imports: [NgSwitch, NgClass, NgSwitchCase, RouterLink, NgSwitchDefault, DaemonNiceNamePipe],
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
}
