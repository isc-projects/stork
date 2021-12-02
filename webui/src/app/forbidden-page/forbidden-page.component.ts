import { Component } from '@angular/core'

/**
 * This component displays HTTP Error 403 page.
 *
 * The page contains the alert that the access to the given
 * page is forbidden. It also provides a link to the Dashboard
 * page.
 */
@Component({
    selector: 'app-forbidden-page',
    templateUrl: './forbidden-page.component.html',
    styleUrls: ['./forbidden-page.component.sass'],
})
export class ForbiddenPageComponent {
    constructor() {}
}
