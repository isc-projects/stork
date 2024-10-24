import { Component, OnInit } from '@angular/core'
import { UntypedFormBuilder, UntypedFormGroup, Validators } from '@angular/forms'
import { Router, ActivatedRoute } from '@angular/router'

import { GeneralService } from '../backend/api/api'
import { AuthenticationMethod } from '../backend/model/authenticationMethod'
import { AuthService } from '../auth.service'
import { HttpClient } from '@angular/common/http'
import { lastValueFrom } from 'rxjs'

/**
 * A component presenting a Stork log in screen.
 *
 * In the simplest form, this component contains a user and password
 * input boxes to log in to the system. However, since Stork can support
 * different authentication methods via hooks, it can also display a
 * customized log in view, depending on the selected method.
 *
 * The log in screen can also be customized for the specific deployments
 * using as static login-screen-welcome.html file. This file can be optionally
 * created and stored in the Stork server's assets/static-page-content/
 * folder. It typically holds the contact information to the system
 * administrators and basic instructions how to obtain access to the system.
 * Read more about it in the Stork ARM.
 */
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

    /**
     * Welcome message fetched.
     *
     * It is fetched from the assets/static-page-content/login-screen-welcome.html
     */
    welcomeMessage: string = null

    constructor(
        protected api: GeneralService,
        private auth: AuthService,
        private route: ActivatedRoute,
        private router: Router,
        private formBuilder: UntypedFormBuilder,
        public http: HttpClient
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
        this.returnUrl = this.route.snapshot.queryParams.returnUrl || '/'

        // Initialize the login form controls.
        this.loginForm = this.formBuilder.group({
            authenticationMethod: ['', Validators.required],
            identifier: ['', Validators.required],
            secret: ['', Validators.required],
        })

        // Check if the welcome message has been configured for the
        // login screen.
        lastValueFrom(this.http.get('assets/static-page-content/login-screen-welcome.html', { responseType: 'text' }))
            .then((data) => {
                // The welcome messages should be brief. Let's avoid bloated
                // welcome messages.
                if (data?.length < 2048) {
                    this.welcomeMessage = data
                }
            })
            .catch((error) => {
                console.log(error)
            })

        // Fetch version.
        lastValueFrom(this.api.getVersion())
            .then((data) => {
                this.version = data.version
            })
            .catch((error) => {
                console.log(error)
            })

        // Fetch authentication methods.
        lastValueFrom(this.auth.getAuthenticationMethods()).then((methods) => {
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
     * Shorthand to get the login form values.
     */
    get f() {
        return this.loginForm.value
    }

    /**
     * Performs a login operation.
     */
    signIn() {
        Object.keys(this.loginForm.controls).forEach((k) => this.loginForm.get(k).markAsDirty())
        if (this.loginForm.valid) {
            this.auth.login(this.f.authenticationMethod.id, this.f.identifier, this.f.secret, this.returnUrl)
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
