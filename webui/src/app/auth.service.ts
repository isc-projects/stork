import { Injectable } from '@angular/core'
import { HttpClient } from '@angular/common/http'
import { Router, ActivatedRoute } from '@angular/router'
import { BehaviorSubject, Observable } from 'rxjs'
import { map } from 'rxjs/operators'

import { MessageService } from 'primeng/api'

import { UsersService } from './backend/api/users.service'

export class User {
    id: number
    username: string
    email: string
    firstName: string
    lastName: string
    groups: number[]
}

export class Group {
    id: number
    name: string
    description: string
}

@Injectable({
    providedIn: 'root',
})
export class AuthService {
    private currentUserSubject: BehaviorSubject<User>
    public currentUser: Observable<User>
    public user: User
    public groups: Group[] = []

    constructor(
        private http: HttpClient,
        private api: UsersService,
        private router: Router,
        private msgSrv: MessageService
    ) {
        this.currentUserSubject = new BehaviorSubject<User>(JSON.parse(localStorage.getItem('currentUser')))
        this.currentUser = this.currentUserSubject.asObservable()
        this.initSystemGroups()
    }

    public get currentUserValue(): User {
        return this.currentUserSubject.value
    }

    initSystemGroups() {
        this.api.getGroups().subscribe(
            data => {
                if (data.items) {
                    for (const i in data.items) {
                        if (data.items.hasOwnProperty(i)) {
                            let group = new Group()
                            group.id = data.items[i].id
                            group.name = data.items[i].name
                            group.description = data.items[i].description
                            this.groups.push(group)
                        }
                    }
                }
            },
            err => {
                this.msgSrv.add({ severity: 'error', summary: 'Unable to fetch user group definitions' })
            }
        )
    }

    login(username: string, password: string, returnUrl: string) {
        let user: User
        this.api.createSession(username, password).subscribe(
            data => {
                if (data.id != null) {
                    user = new User()

                    user.id = data.id
                    user.username = data.login
                    user.email = data.email
                    user.firstName = data.name
                    user.lastName = data.lastname

                    // Store groups the user belongs to.
                    user.groups = []
                    for (const i in data.groups) {
                        if (data.groups.hasOwnProperty(i)) {
                            user.groups.push(data.groups[i])
                        }
                    }

                    this.currentUserSubject.next(user)
                    localStorage.setItem('currentUser', JSON.stringify(user))
                    this.router.navigate([returnUrl])
                }
            },
            err => {
                this.msgSrv.add({ severity: 'error', summary: 'Invalid login or password' })
            }
        )
        return user
    }

    logout() {
        this.api.deleteSession('response').subscribe(resp => {
            localStorage.removeItem('currentUser')
            this.currentUserSubject.next(null)
        })
    }

    superAdmin(): boolean {
        if (this.currentUserValue && this.currentUserValue.groups) {
            for (const i in this.currentUserValue.groups) {
                if (this.currentUserValue.groups[i] === 1) {
                    return true
                }
            }
        }
        return false
    }
}
