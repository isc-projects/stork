import { Component, OnInit } from '@angular/core'
import { MenuItem } from 'primeng/api'

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

    constructor() {}

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
                    {
                        label: 'Change password',
                        id: 'change-password',
                        icon: 'pi pi-lock',
                        routerLink: '/profile/password',
                    },
                ],
            },
        ]
    }
}
