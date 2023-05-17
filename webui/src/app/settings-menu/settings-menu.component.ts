import { Component, OnInit } from '@angular/core'
import { MenuItem } from 'primeng/api'
import { AuthService } from '../auth.service'

/**
 * This component provides a menu for navigating between different
 * settings of a logged user. Currently supported operations are
 * to display information about the user and modifying user's
 * password.
 */
@Component({
    selector: 'app-settings-menu',
    templateUrl: './settings-menu.component.html',
    styleUrls: ['./settings-menu.component.sass'],
})
export class SettingsMenuComponent implements OnInit {
    items: MenuItem[]

    constructor(public auth: AuthService) {}

    /**
     * Initializes the menu items.
     */
    ngOnInit() {
        this.items = [
            {
                label: 'User Profile',
                items: [
                    {
                        label: 'Settings',
                        id: 'user-settings',
                        icon: 'pi pi-user',
                        routerLink: '/profile/settings',
                    },
                ],
            },
        ]

        if (this.auth.isInternalUser()) {
            // Only users authenticated using the credentials stored in the Stork
            // database can change the password using the Stork UI.
            // TODO: This button should always be available but the external
            // users should be redirected to the external change password endpoint.
            this.items[0].items.push({
                label: 'Change password',
                id: 'change-password',
                icon: 'pi pi-lock',
                routerLink: '/profile/password',
            })
        }
    }
}
