import { Component, OnInit } from '@angular/core'

@Component({
    selector: 'app-settings-menu',
    templateUrl: './settings-menu.component.html',
    styleUrls: ['./settings-menu.component.sass'],
})
export class SettingsMenuComponent implements OnInit {
    items: MenuItem[]

    constructor() {}

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
