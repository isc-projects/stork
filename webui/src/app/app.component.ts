import { Component, OnInit } from '@angular/core'
import { Router } from '@angular/router'
import { Observable } from 'rxjs'

import { MenubarModule } from 'primeng/menubar'
import { MenuItem } from 'primeng/api'

import { AuthService, User } from './auth.service'
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
                label: 'Configuration',
                items: [
                    {
                        label: 'Users',
                        icon: 'fa fa-user',
                        routerLink: '/users',
                    },
                ],
            },
            {
                label: 'Services',
                items: [
                    {
                        label: 'Machines',
                        icon: 'fa fa-server',
                        routerLink: '/machines/all',
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
