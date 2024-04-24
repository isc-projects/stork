import { Component, OnInit } from '@angular/core'
import { UntypedFormBuilder, UntypedFormGroup, Validators } from '@angular/forms'
import { Router, ActivatedRoute } from '@angular/router'

import { GeneralService } from '../backend/api/api'
import { AuthenticationMethod } from '../backend/model/authenticationMethod'
import { AuthService } from '../auth.service'

@Component({
    selector: 'app-login-screen',
    templateUrl: './login-screen.component.html',
    styleUrls: ['./login-screen.component.sass'],
})
export class LoginScreenComponent implements OnInit {
    /**
     * Stork version.
     */
    version = 'not available'

    /**
     * The URL address redirected from.
     */
    returnUrl: string

    /**
     * Object representing the login form.
     */
    loginForm: UntypedFormGroup

    /**
     * List of available authentication methods.
     */
    authenticationMethods: AuthenticationMethod[]

    /**
     * Selected authentication method.
     */
    authenticationMethod: AuthenticationMethod

    constructor(
        protected api: GeneralService,
        private auth: AuthService,
        private route: ActivatedRoute,
        private router: Router,
        private formBuilder: UntypedFormBuilder
    ) {}

    /**
     * Fetches the version and authentication methods, and initializes the
     * login form.
     *
     * If the current URL is logout, immediately performs the log out operation.
     */
    ngOnInit() {
        if (this.router.url === '/logout') {
            this.signOut()
        }

        // Set the return URL.
        let returnUrl: string = this.route.snapshot.queryParams.returnUrl || '/'
        if (!returnUrl.startsWith('/')) {
            returnUrl = '/' + returnUrl
        }
        this.returnUrl = returnUrl

        // Initialize the login form controls.
        this.loginForm = this.formBuilder.group({
            authenticationMethod: ['', Validators.required],
            identifier: ['', Validators.required],
            secret: ['', Validators.required],
        })

        // Fetch version.
        this.api
            .getVersion()
            .toPromise()
            .then((data) => {
                this.version = data.version
            })
            .catch((error) => {
                console.log(error)
            })

        // Fetch authentication methods.
        this.auth
            .getAuthenticationMethods()
            .toPromise()
            .then((methods) => {
                this.authenticationMethods = methods
                this.authenticationMethod = methods[0]
                this.loginForm.controls.authenticationMethod.setValue(this.authenticationMethod)
            })
    }

    /**
     * Callback called when the authentication method icon is missing.
     * It hides the icon to prevent displaying an ugly placeholder.
     * The target element is the image (<img>) element.
     *
     * @param ev A loading event.
     */
    onMissingIcon(ev: Event) {
        const targetElement = ev.target as HTMLElement
        targetElement.classList.add('hidden')
    }

    /**
     * Shorthand to get the login form controls.
     */
    get f() {
        return this.loginForm.controls
    }

    /**
     * Performs a login operation.
     */
    signIn() {
        Object.keys(this.loginForm.controls).forEach((k) => this.loginForm.get(k).markAsDirty())
        if (this.loginForm.valid) {
            this.auth.login(this.authenticationMethod.id, this.f.identifier.value, this.f.secret.value, this.returnUrl)
            this.router.navigate([this.returnUrl])
        }
    }

    /**
     * Performs a log out operation.
     * It redirect a user to the login form.
     */
    signOut() {
        this.auth.logout()
        this.router.navigate(['/login'])
    }

    /**
     * Callback called on the key pressing.
     * It triggers the login operation if the Enter key is pressed.
     * @param event
     */
    keyUp(event) {
        if (event.key === 'Enter') {
            this.signIn()
        }
    }
}
