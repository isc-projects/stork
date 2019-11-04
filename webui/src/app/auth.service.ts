import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router, ActivatedRoute } from '@angular/router';
import { BehaviorSubject, Observable } from 'rxjs';
import { map } from 'rxjs/operators';

import {MessageService} from 'primeng/api';

import { ServicesService } from './backend/api/services.service';

export class User {
    id: number;
    username: string;
    email: string;
    firstName: string;
    lastName: string;

    public get identifier(): string {
        return this.username || this.email
    }
}

@Injectable({
  providedIn: 'root'
})
export class AuthService {
    private currentUserSubject: BehaviorSubject<User>;
    public currentUser: Observable<User>;
    public user: User;

    constructor(private http: HttpClient, private api: ServicesService, private router: Router) {
        this.currentUserSubject = new BehaviorSubject<User>(JSON.parse(localStorage.getItem('currentUser')));
        this.currentUser = this.currentUserSubject.asObservable();
    }

    public get currentUserValue(): User {
        return this.currentUserSubject.value;
    }

    login(username: string, password: string, returnUrl: string, msgSrv: MessageService) {
        let user: User;
        this.api.sessionsPost(username, password).subscribe(data => {
            if (data.id != null) {
                user = new User();

                user.id = data.id;
                user.username = data.login;
                user.email = data.email;
                user.firstName = data.firstname;
                user.lastName = data.lastname;
                this.currentUserSubject.next(user);
                localStorage.setItem('currentUser', JSON.stringify(user));
                this.router.navigate([returnUrl]);
            }
        },
        err => {
            msgSrv.add({severity: 'error', summary: 'Invalid login or password'});
        });
        return user;
    }

    logout() {
        this.api.sessionsDelete('response').subscribe(resp => {
            localStorage.removeItem('currentUser');
            this.currentUserSubject.next(null);
        });
    }
}
