import { Component } from '@angular/core'

@Component({
    selector: 'app-config-checker-preference-page',
    templateUrl: './config-checker-preference-page.component.html',
    styleUrls: ['./config-checker-preference-page.component.sass'],
})
export class ConfigCheckerPreferencePageComponent {
    breadcrumbs = [{ label: 'Configuration' }, { label: 'Review checkers' }]

    constructor() {}
}
