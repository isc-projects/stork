import { Component, OnInit } from '@angular/core'

@Component({
    selector: 'app-config-checker-preference-page',
    templateUrl: './config-checker-preference-page.component.html',
    styleUrls: ['./config-checker-preference-page.component.sass'],
})
export class ConfigCheckerPreferencePageComponent implements OnInit {
    breadcrumbs = [{ label: 'Configuration' }, { label: 'Review checkers' }]

    constructor() {}

    ngOnInit(): void {}
}
