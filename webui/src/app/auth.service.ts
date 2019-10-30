import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Router, ActivatedRoute } from '@angular/router';
import { BehaviorSubject, Observable } from 'rxjs';
import { map } from 'rxjs/operators';

import { DefaultService } from './backend/api/default.service';

export class User {
    id: number;
    username: string;
    email: string;
    firstName: string;
    lastName: string;
}

@Injectable({
  providedIn: 'root'
})
export class AuthService {
    private currentUserSubject: BehaviorSubject<User>;
    public currentUser: Observable<User>;
    public user: User;

    constructor(private http: HttpClient, private api: DefaultService, private router: Router) {
        this.currentUserSubject = new BehaviorSubject<User>(JSON.parse(localStorage.getItem('currentUser')));
        this.currentUser = this.currentUserSubject.asObservable();
    }

    public get currentUserValue(): User {
        return this.currentUserSubject.value;
    }

    login(username: string, password: string) {
        var user: User;
        this.api.sessionPost(username, password, 'body').subscribe(data => {
            if (data.id != null) {
                user = new User();

                user.id = data.id;
                user.username = data.login;
                user.email = data.email;
                user.firstName = data.firstname;
                user.lastName = data.lastname;
                this.currentUserSubject.next(user);
                localStorage.setItem('currentUser', JSON.stringify(user))
                this.router.navigate(["/"])
                
            }
        });
        // return this.http.post<any>(`${environment.apiUrl}/users/authenticate`, { username, password })
        //     .pipe(map(user => {
        //         // login successful if there's a jwt token in the response
        //         if (user && user.token) {
        //             // store user details and jwt token in local storage to keep user logged in between page refreshes
        //             localStorage.setItem('currentUser', JSON.stringify(user));
        //             this.currentUserSubject.next(user);
        //         }

        //         return user;
        //     }));
        return user;
    }

    logout() {
        // remove user from local storage to log user out
        localStorage.removeItem('currentUser');
        this.currentUserSubject.next(null);
    }
}
