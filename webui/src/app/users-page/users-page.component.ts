import { Component, OnInit } from '@angular/core'

import { UsersService } from '../backend/api/api'

@Component({
    selector: 'app-users-page',
    templateUrl: './users-page.component.html',
    styleUrls: ['./users-page.component.sass'],
})
export class UsersPageComponent implements OnInit {
    // users table
    users: any[]
    totalUsers: number

    constructor(private usersApi: UsersService) {}

    ngOnInit() {
        this.users = []
    }

    loadUsers(event) {
        this.usersApi.getUsers(event.first, event.rows, event.filters.text).subscribe(data => {
            this.users = data.items
            this.totalUsers = data.total
        })
    }
}
