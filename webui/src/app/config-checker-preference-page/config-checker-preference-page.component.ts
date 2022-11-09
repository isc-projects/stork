import { Component } from '@angular/core'

/**
 * Displays the global settings page of the configuration review checkers.
 * It contains the checkers' picker and some help text.
 */
@Component({
    selector: 'app-config-checker-preference-page',
    templateUrl: './config-checker-preference-page.component.html',
    styleUrls: ['./config-checker-preference-page.component.sass'],
})
export class ConfigCheckerPreferencePageComponent {
    breadcrumbs = [{ label: 'Configuration' }, { label: 'Review Checkers' }]

    constructor() {}
}
