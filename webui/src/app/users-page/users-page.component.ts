import { Component, OnInit } from '@angular/core';

@Component({
    selector: 'app-users-page',
    templateUrl: './users-page.component.html',
    styleUrls: ['./users-page.component.sass']
})
export class UsersPageComponent implements OnInit {
    // users table
    users: any[]
    totalUsers: number

    constructor() { }

    ngOnInit() {
        this.users = []
    }

    loadUsers(event) {
        this.usersApi.getUsers().subscribe(data => {
            this.users = data.items
            this.totalUsers = data.total
        })
    }

}
