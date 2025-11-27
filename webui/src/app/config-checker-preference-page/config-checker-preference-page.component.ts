import { Component } from '@angular/core'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { ConfigCheckerPreferenceUpdaterComponent } from '../config-checker-preference-updater/config-checker-preference-updater.component'

/**
 * Displays the global settings page of the configuration review checkers.
 * It contains the checkers' picker and some help text.
 */
@Component({
    selector: 'app-config-checker-preference-page',
    templateUrl: './config-checker-preference-page.component.html',
    styleUrls: ['./config-checker-preference-page.component.sass'],
    imports: [BreadcrumbsComponent, ConfigCheckerPreferenceUpdaterComponent],
})
export class ConfigCheckerPreferencePageComponent {
    breadcrumbs = [{ label: 'Configuration' }, { label: 'Review Checkers' }]

    constructor() {}
}
