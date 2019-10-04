import { Component } from '@angular/core';
import { Router } from '@angular/router';

import {MenubarModule} from 'primeng/menubar';
import {MenuItem} from 'primeng/api';

import { AuthService, User } from './auth.service';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.sass']
})
export class AppComponent {
    title = 'Stork';
    currentUser = null;

    items: MenuItem[];

    constructor(
        private router: Router,
        private auth: AuthService
    ) {
        this.auth.currentUser.subscribe(x => this.currentUser = x);
    }

    ngOnInit() {
        this.items = [
            {
                label: 'DHCP',
            },
            {
                label: 'DNS',
            }
        ];
    }

    signOut() {
        this.auth.logout();
        this.router.navigate(['/login']);
    }
}
