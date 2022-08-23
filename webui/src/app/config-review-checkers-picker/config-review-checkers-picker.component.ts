import { Component, EventEmitter, Input, OnInit, Output } from '@angular/core'
import { ConfigChecker, ConfigCheckerPreference, Configuration } from "../backend"

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
    @Output() onChangePreference = new EventEmitter<ConfigCheckerPreference>()

    /**
     * Use tri-state checkboxes to specify the checker state
     */
    @Input() allowInheritState: boolean = false

    private _getNextState(state: ConfigChecker.StateEnum): ConfigChecker.StateEnum {
        if (state === ConfigChecker.StateEnum.Inherit) {
            return ConfigChecker.StateEnum.Enabled
        } else if (state === ConfigChecker.StateEnum.Enabled) {
            return ConfigChecker.StateEnum.Disabled
        } else {
            if (this.allowInheritState) {
                return ConfigChecker.StateEnum.Inherit
            } else {
                return ConfigChecker.StateEnum.Enabled
            }
        }
    }

    getTriggerIcon(trigger: string): string {
        switch (trigger) {
            case "internal":
                return "fa fa-eye-slash"
            case "manual":
                return "fa fa-hand-paper"
            case "config change":
                return "fa fa-tools"
            case "host reservation change":
                return "fa fa-registered"
            default:
                return null
        }
    }

    getSelectorIcon(selector: string): string {
        switch (selector) {
            case "each-daemon":
                return "fa fa-dice-d20"
            case "kea-daemon":
                return "fa fa-dice-d6"
            case "kea-ca-daemon":
                return "fa fa-cube"
            case "kea-dhcp-daemon":
                return "fa fa-dice"
            case "kea-dhcp-v4-daemon":
                return "fa fa-dice-four"
            case "kea-dhcp-v6-daemon":
                return "fa fa-dice-six"
            case "kea-d2-daemon":
                return "fa fa-dice-two"
            case "bind9-daemon":
                return "fa fa-dot-circle"
            default:
                return null
        }
    }

    onCheckerPreferenceChange(event, checker: ConfigChecker) {
        checker.state = this._getNextState(checker.state)
        this.onChangePreference.emit({
            name: checker.name,
            state: checker.state
        })
    }
}