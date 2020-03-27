import { Component, OnInit } from '@angular/core'
import { Router } from '@angular/router'
import { Observable } from 'rxjs'

import { MenuItem } from 'primeng/api'

import { GeneralService } from './backend/api/api'
import { AuthService } from './auth.service'
import { LoadingService } from './loading.service'

@Component({
    selector: 'app-root',
    templateUrl: './app.component.html',
    styleUrls: ['./app.component.sass'],
})
export class AppComponent implements OnInit {
    title = 'Stork'
    storkVersion = 'unknown'
    storkBuildDate = 'unknown'
    currentUser = null
    loadingInProgress = new Observable()

    menuItems: MenuItem[]

    constructor(
        private router: Router,
        protected generalApi: GeneralService,
        private auth: AuthService,
        private loadingService: LoadingService
    ) {
        this.auth.currentUser.subscribe(x => {
            this.currentUser = x
            this.initMenuItems()
        })
        this.loadingInProgress = this.loadingService.getState()
    }

    initMenuItems() {
        this.menuItems = []
        this.menuItems.push({
            label: 'DHCP',
            items: [
                {
                    label: 'Hosts',
                    icon: 'fa fa-laptop',
                    routerLink: '/dhcp/hosts',
                },
                {
                    label: 'Subnets',
                    icon: 'fa fa-project-diagram',
                    routerLink: '/dhcp/subnets',
                },
                {
                    label: 'Shared Networks',
                    icon: 'fa fa-network-wired',
                    routerLink: '/dhcp/shared-networks',
                },
            ],
        })
        this.menuItems.push({
            label: 'Services',
            items: [
                {
                    label: 'Kea DHCP',
                    icon: 'fa fa-server',
                    routerLink: '/apps/kea/all',
                },
                {
                    label: 'BIND 9 DNS',
                    icon: 'fa fa-server',
                    routerLink: '/apps/bind9/all',
                },
                {
                    label: 'Machines',
                    icon: 'fa fa-server',
                    routerLink: '/machines/all',
                },
            ],
        })
        if (this.auth.superAdmin()) {
            this.menuItems = this.menuItems.concat([
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
            ])
        }
        this.menuItems.push({
            label: 'Profile',
            items: [
                {
                    label: 'Settings',
                    icon: 'fa fa-cog',
                    routerLink: '/settings',
                },
                {
                    label: 'Logout',
                    icon: 'pi pi-sign-out',
                    routerLink: '/logout',
                },
            ],
        })
        this.menuItems.push({
            label: 'Help',
            items: [
                {
                    label: 'Manual',
                    icon: 'fa fa-book',
                    url: '/assets/arm/index.html',
                    target: 'blank',
                },
                {
                    label: 'API Docs (SwaggerUI)',
                    icon: 'fa fa-code',
                    routerLink: '/swagger-ui',
                },
                {
                    label: 'API Docs (Redoc)',
                    icon: 'fa fa-code',
                    url: '/api/docs',
                    target: 'blank',
                },
            ],
        })
    }

    ngOnInit() {
        this.initMenuItems()

        this.generalApi.getVersion().subscribe(data => {
            this.storkVersion = data.version
            this.storkBuildDate = data.date
        })
    }

    signOut() {
        this.router.navigate(['/logout'])
    }
}
