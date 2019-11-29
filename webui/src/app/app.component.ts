import { Component, OnInit } from '@angular/core'
import { Router } from '@angular/router'
import { Observable } from 'rxjs'

import { MenuItem } from 'primeng/api'

import { AuthService } from './auth.service'
import { LoadingService } from './loading.service'

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.sass'],
})
export class AppComponent implements OnInit {
    title = 'Stork'
    currentUser = null
    loadingInProgress = new Observable()

    menuItems: MenuItem[]

    constructor(private router: Router, private auth: AuthService, private loadingService: LoadingService) {
        this.auth.currentUser.subscribe(x => (this.currentUser = x))
        this.loadingInProgress = this.loadingService.getState()
    }

    ngOnInit() {
        this.menuItems = [
            {
                label: 'Services',
                items: [
                    {
                        label: 'Kea DHCP',
                        icon: 'fa fa-server',
                        routerLink: '/services/kea/all',
                    },
                    // TODO: add support for BIND services
                    // {
                    //     label: 'BIND DNS',
                    //     icon: 'fa fa-server',
                    //     routerLink: '/services/bind/all',
                    // },
                    {
                        label: 'Machines',
                        icon: 'fa fa-server',
                        routerLink: '/machines/all',
                    },
                ],
            },
            {
                label: 'Configuration',
                items: [
                    {
                        label: 'Users',
                        icon: 'fa fa-user',
                        routerLink: '/users',
                    },
                ],
            },
        ]
    }

    signOut() {
        this.auth.logout()
        this.router.navigate(['/login'])
    }
}
