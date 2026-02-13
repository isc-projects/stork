import { Component, input, Input } from '@angular/core'
import { NgSwitch, NgClass, NgSwitchCase, NgSwitchDefault, NgIf } from '@angular/common'
import { RouterLink } from '@angular/router'
import { DaemonNiceNamePipe } from '../pipes/daemon-name.pipe'

/**
 * A component that displays given entity as a link with rounded border
 * and a background color. It can be used for making a link to:
 * - machine
 * - daemon
 * - subnet
 * - host reservation
 * - user
 * - shared network
 */
@Component({
    selector: 'app-entity-link',
    templateUrl: './entity-link.component.html',
    styleUrls: ['./entity-link.component.sass'],
    imports: [NgSwitch, NgIf, NgClass, NgSwitchCase, RouterLink, NgSwitchDefault, DaemonNiceNamePipe],
})
export class EntityLinkComponent {
    /**
     * Entity name, one of: machine, daemon, subnet, host.
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

    /**
     * Input boolean flag controlling whether subnet, host reservation or shared network entity link
     * should include the entity identifier as well.
     * For daemon, machine or user entity, the identifier is always included.
     * It defaults to false.
     */
    showIdentifier = input<boolean>(false)

    constructor() {}
}
