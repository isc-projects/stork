import { Component, OnDestroy, OnInit } from '@angular/core'
import { UntypedFormBuilder, UntypedFormGroup, Validators } from '@angular/forms'
import { Router, ActivatedRoute } from '@angular/router'
import { HttpResponse } from '@angular/common/http'

import { ButtonModule } from 'primeng/button'

import { GeneralService } from '../backend/api/api'
import { AuthenticationMethod } from '../backend/model/authenticationMethod'
import { AuthService } from '../auth.service'
import { Subscription } from 'rxjs'

@Component({
    selector: 'app-login-screen',
    templateUrl: './login-screen.component.html',
    styleUrls: ['./login-screen.component.sass'],
})
export class LoginScreenComponent implements OnInit, OnDestroy {
    private subscriptions = new Subscription()
    version = 'not available'
    returnUrl: string
    loginForm: UntypedFormGroup
    authenticationMethods: AuthenticationMethod[]
    authenticationMethod: AuthenticationMethod

    constructor(
        protected api: GeneralService,
        private auth: AuthService,
        private route: ActivatedRoute,
        private router: Router,
        private formBuilder: UntypedFormBuilder
    ) {}

    ngOnInit() {
        if (this.router.url === '/logout') {
            this.signOut()
        }

        this.returnUrl = this.route.snapshot.queryParams.returnUrl || '/'

        this.loginForm = this.formBuilder.group({
            authenticationMethod: ['', Validators.required],
            identifier: ['', Validators.required],
            secret: ['', Validators.required],
        })

        this.subscriptions.add(
            this.api.getVersion().subscribe(
                (data) => {
                    console.info(data)
                    this.version = data.version
                },
                (error) => {
                    console.log(error)
                }
            )
        )

        this.subscriptions.add(
            this.auth.getAuthenticationMethods().subscribe((methods) => {
                this.authenticationMethods = methods
                this.authenticationMethod = methods[0]
            })
        )
    }

    onMissingIcon(ev: Event) {
        ;(ev.target as HTMLElement).classList.add('hidden')
    }

    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    get f() {
        return this.loginForm.controls
    }

    signIn() {
        this.performLogin()
    }

    signOut() {
        this.auth.logout()
        this.router.navigate(['/login'])
    }

    keyUp(event) {
        if (event.key === 'Enter') {
            this.performLogin()
        }
    }

    private performLogin() {
        this.auth.login(this.authenticationMethod.id, this.f.identifier.value, this.f.secret.value, this.returnUrl)
        this.router.navigate([this.returnUrl])
    }
}
