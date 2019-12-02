import { Component, OnInit } from '@angular/core'
import { FormBuilder, FormGroup, Validators } from '@angular/forms'
import { Router, ActivatedRoute } from '@angular/router'
import { HttpResponse } from '@angular/common/http'

import { ButtonModule } from 'primeng/button'

import { GeneralService } from '../backend/api/api'
import { AuthService } from '../auth.service'

@Component({
    selector: 'app-login-screen',
    templateUrl: './login-screen.component.html',
    styleUrls: ['./login-screen.component.sass'],
})
export class LoginScreenComponent implements OnInit {
    version = 'not available'
    returnUrl: string
    loginForm: FormGroup

    constructor(
        protected api: GeneralService,
        private auth: AuthService,
        private route: ActivatedRoute,
        private router: Router,
        private formBuilder: FormBuilder
    ) {}

    ngOnInit() {
        if (this.router.url === '/logout') {
            this.signOut()
        }

        this.returnUrl = this.route.snapshot.queryParams.returnUrl || '/'

        this.loginForm = this.formBuilder.group({
            username: ['', Validators.required],
            password: ['', Validators.required],
        })

        this.api.getVersion().subscribe(data => {
            console.info(data)
            this.version = data.version
        })
    }

    get f() {
        return this.loginForm.controls
    }

    signIn() {
        this.auth.login(this.f.username.value, this.f.password.value, this.returnUrl)
        this.router.navigate([this.returnUrl])
    }

    signOut() {
        this.auth.logout()
        this.router.navigate(['/login'])
    }
}
