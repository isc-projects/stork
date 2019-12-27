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
                label: 'User Settings',
                items: [
                    {
                        label: 'Profile',
                        icon: 'pi pi-user',
                        routerLink: '/settings/profile',
                    },
                    {
                        label: 'Change password',
                        icon: 'pi pi-lock',
                        routerLink: '/settings/password',
                    },
                ],
            },
        ]
    }
}
