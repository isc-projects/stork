import { Component, EventEmitter, Input, Output } from '@angular/core'
import { ConfigChecker, ConfigCheckerPreference } from '../backend'

@Component({
    selector: 'app-config-checker-preference-picker',
    templateUrl: './config-checker-preference-picker.component.html',
    styleUrls: ['./config-checker-preference-picker.component.sass'],
})
export class ConfigCheckerPreferencePicker {
    /**
     * List of the config checkers.
     */
    @Input() checkers: ConfigChecker[]

    /**
     * Stream of the changed config checker preferences.
     */
    @Output() changePreference = new EventEmitter<ConfigCheckerPreference>()

    /**
     * Use tri-state checkboxes to specify the checker state
     */
    @Input() allowInheritState: boolean = false

    /**
     * It cycles the checker states. The order is enabled - disabled - inherit.
     * It skips the inherit state if the component is configured to disallow it.
     * @param state Checker state
     * @returns Next checker state
     */
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

    /**
     * Returns a fancy icon for the checker trigger. If the trigger is unknown
     * then returns no icon.
     * @param trigger The checker trigger
     * @returns Icon CSS classes
     */
    getTriggerIcon(trigger: string): string {
        switch (trigger) {
            case 'internal':
                return 'fa fa-eye-slash'
            case 'manual':
                return 'fa fa-hand-paper'
            case 'config change':
                return 'fa fa-tools'
            case 'host reservation change':
                return 'fa fa-registered'
            default:
                return null
        }
    }

    /**
     * Returns a fancy icon for the checker selector. If the selector is unknown
     * then returns no icon.
     * We don't have specialized icons for our daemons, and FontAwesome doesn't
     * contain any icons related to DHCP or DNS. But the chips with icons look
     * better than those without. The dices aren't the first thing you associate
     * with the Stork-supported daemons, but it has a little sense:
     * - Kea DHCPv4 is a die with a single 4-dots-side visible
     * - Kea DHCPv6 is a die with a single 6-dots-side visible
     * - Kea D2 daemon is a die with a single 2-dots-side visible
     * - General DHCP daemon is two dice with a single side visible
     * - General Kea daemon is a 6-side die in isometric projection
     * - Kea Control daemon is a 6-side die in isometric projection with
     *   highlighted one side
     * - General daemon is a fancy representation of 20-side die
     * - Bind 9 daemon is a circle with a single dot in the center because it
     *   has a dot similar to Kea DHCP dice (Bind 9 is a specific daemon), but
     *   the circle is opposite of a square (DNS is an entirely different thing
     *   than DHCP). Additionally, the circle is similar to the 20-side dice in
     *   the same way as the square. (20-side dice is a generalization of daemon).
     * @param selector The checker selector
     * @returns Icon CSS classes
     */
    getSelectorIcon(selector: string): string {
        switch (selector) {
            case 'each-daemon':
                return 'fa fa-dice-d20'
            case 'kea-daemon':
                return 'fa fa-dice-d6'
            case 'kea-ca-daemon':
                return 'fa fa-cube'
            case 'kea-dhcp-daemon':
                return 'fa fa-dice'
            case 'kea-dhcp-v4-daemon':
                return 'fa fa-dice-four'
            case 'kea-dhcp-v6-daemon':
                return 'fa fa-dice-six'
            case 'kea-d2-daemon':
                return 'fa fa-dice-two'
            case 'bind9-daemon':
                return 'fa fa-dot-circle'
            default:
                return null
        }
    }

    getCheckerDescription(checkerName: string): string {
        switch (checkerName) {
            case "stat_cmds_presence":
                return "The checker verifying if the stat_cmds hooks library is loaded."
            case "host_cmds_presence":
                return "The checker verifying if the host_cmds hooks library is "
                    + "loaded when host backend is in use."
            case "shared_network_dispensable":
                return "The checker verifying if a shared network can be removed "
                    + "because it is empty or contains only one subnet."
            case "subnet_dispensable":
                return "The checker verifying if a subnet can be removed because "
                    + "it includes no pools and no reservations. The check is "
                    + "skipped when the host_cmds hook library is loaded because "
                    + "host reservations may be present in the database."
            case "reservations_out_of_pool":
                return "The checker suggesting the use of out-of-pool host "
                    + "reservation mode when there are subnets with all host "
                    + "reservations outside of the dynamic pools."
            default:
                return ""
        }
    }

    /**
     * Callback called on change the checker state. It emits an Angular event
     * with changed checker preference.
     * @param event Generic change input event
     * @param checker Affected checker
     */
    onCheckerStateChange(event, checker: ConfigChecker) {
        checker.state = this._getNextState(checker.state)
        this.changePreference.emit({
            name: checker.name,
            state: checker.state,
        })
    }
}
