import { Component, OnInit, Input } from '@angular/core'

/**
 * A component that displays given entity as a link with rounded border
 * and a background color. It can be used for making a link to:
 * - machine
 * - app
 * - daemon
 * - subnet
 */
@Component({
    selector: 'app-entity-link',
    templateUrl: './entity-link.component.html',
    styleUrls: ['./entity-link.component.sass'],
})
export class EntityLinkComponent implements OnInit {
    // Entity name, one of: machine, app, daemon, subnet.
    @Input() entity: string

    // Attributes that describe given entity e.g. id, name, etc.
    @Input() attrs: any

    constructor() {}

    ngOnInit(): void {}
}
