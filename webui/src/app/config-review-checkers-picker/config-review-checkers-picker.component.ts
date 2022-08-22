import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core'
import { ConfigChecker, ConfigCheckerPreference } from "../backend"

@Component({
    selector: 'app-config-review-checkers-picker',
    templateUrl: './config-review-checkers-picker.component.html',
    styleUrls: ['./config-review-checkers-picker.component.sass'],
})
export class ConfigReviewCheckersPickerComponent {

    /**
     * List of the config checkers.
     */
    @Input() checkers: ConfigChecker[]

    /**
     * Stream of the changed config checker preferences.
     */
    @Output() changedPreferences = new EventEmitter<ConfigCheckerPreference>()
}