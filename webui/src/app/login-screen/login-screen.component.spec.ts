import { ComponentFixture, fakeAsync, TestBed, tick, waitForAsync } from '@angular/core/testing'

import { LoginScreenComponent } from './login-screen.component'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { AuthenticationMethod, GeneralService, UsersService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { SelectButtonModule } from 'primeng/selectbutton'
import { ButtonModule } from 'primeng/button'
import { of } from 'rxjs'
import { By } from '@angular/platform-browser'
import { RouterModule } from '@angular/router'
import { MessagesModule } from 'primeng/messages'
import { AuthService } from '../auth.service'
import { DropdownModule } from 'primeng/dropdown'
import { PasswordModule } from 'primeng/password'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'

describe('LoginScreenComponent', () => {
    let component: LoginScreenComponent
    let fixture: ComponentFixture<LoginScreenComponent>
    let authServiceStub: Partial<AuthService>

    beforeEach(waitForAsync(() => {
        authServiceStub = {
            getAuthenticationMethods: () =>
                of([
                    {
                        id: 'localId',
                        name: 'local',
                        description: 'local description',
                        formLabelIdentifier: 'localLabelId',
                        formLabelSecret: 'localLabelSecret',
                    },
                    {
                        id: 'ldapId',
                        name: 'ldap',
                        description: 'ldap description',
                        formLabelIdentifier: 'ldapLabelId',
                        formLabelSecret: 'ldapLabelSecret',
                    },
                ] as AuthenticationMethod[]),
            login: () => ({
                id: 1,
                authenticationMethodId: 'ldap',
            }),
        }
        TestBed.configureTestingModule({
            imports: [
                ReactiveFormsModule,
                FormsModule,
                RouterModule.forRoot([]),
                HttpClientTestingModule,
                ProgressSpinnerModule,
                SelectButtonModule,
                ButtonModule,
                MessagesModule,
                DropdownModule,
                PasswordModule,
                BrowserAnimationsModule,
            ],
            declarations: [LoginScreenComponent],
            providers: [
                GeneralService,
                UsersService,
                MessageService,
                { provide: AuthService, useValue: authServiceStub },
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(LoginScreenComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display welcome message', fakeAsync(() => {
        spyOn(component.http, 'get').and.returnValue(of('This is a welcome message'))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        const welcomeMessage = fixture.debugElement.query(By.css('.login-screen__welcome'))
        expect(welcomeMessage).toBeTruthy()
        expect(welcomeMessage.nativeElement.innerText).toContain('This is a welcome message')
    }))

    it('should not display bloated welcome message', fakeAsync(() => {
        spyOn(component.http, 'get').and.returnValue(of('a'.repeat(2049)))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        const welcomeMessage = fixture.debugElement.query(By.css('.login-screen__welcome'))
        expect(welcomeMessage).toBeFalsy()
    }))

    it('should display authentication methods', fakeAsync(() => {
        // Inject AuthService stub.
        fixture.debugElement.injector.get(AuthService)
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        // Check if data was received from AuthService getAuthenticationMethods().
        expect(component.authenticationMethods).toBeTruthy()
        expect(component.authenticationMethods.length).toEqual(2)

        // There should be a dropdown visible.
        const dropdown = fixture.debugElement.query(By.css('.login-screen__authentication-selector .p-dropdown'))
        expect(dropdown).toBeTruthy()

        dropdown.nativeElement.click()
        fixture.detectChanges()

        // Dropdown should display two methods.
        const listItems = dropdown.queryAll(By.css('.p-dropdown-panel li'))
        expect(listItems).toBeTruthy()
        expect(listItems.length).toEqual(2)
        expect(listItems[0].nativeElement.innerText).toContain('local')
        expect(listItems[1].nativeElement.innerText).toContain('ldap')
    }))

    it('should try to sign-in user with selected authentication method', fakeAsync(() => {
        // Inject AuthService stub and set spy.
        const authService = fixture.debugElement.injector.get(AuthService)
        const loginSpy = spyOn(authService, 'login').and.callThrough()
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        // There should be a dropdown visible.
        const dropdown = fixture.debugElement.query(By.css('.login-screen__authentication-selector .p-dropdown'))
        expect(dropdown).toBeTruthy()

        dropdown.nativeElement.click()
        fixture.detectChanges()

        // Let's pick ldap method.
        const listItems = dropdown.queryAll(By.css('li'))
        expect(listItems).toBeTruthy()
        expect(listItems.length).toEqual(2)
        expect(listItems[0].nativeElement.innerText).toContain('local')
        expect(listItems[1].nativeElement.innerText).toContain('ldap')

        listItems[1].nativeElement.click()
        tick()
        fixture.detectChanges()

        // Provide login and password.
        const inputs = fixture.debugElement.queryAll(By.css('.login-screen__authentication-inputs input'))
        expect(inputs).toBeTruthy()
        expect(inputs.length).toEqual(2)

        inputs[0].nativeElement.value = 'login'
        inputs[0].nativeElement.dispatchEvent(new Event('input'))
        fixture.detectChanges()

        inputs[1].nativeElement.value = 'passwd'
        inputs[1].nativeElement.dispatchEvent(new Event('input'))
        fixture.detectChanges()

        // Click Sign In.
        const btn = fixture.debugElement.query(By.css('.login-screen__authentication-inputs button'))
        expect(btn).toBeTruthy()
        btn.nativeElement.click()
        fixture.detectChanges()

        // Check if AuthService login() was called with expected values.
        expect(loginSpy).toHaveBeenCalledOnceWith('ldapId', 'login', 'passwd', '/')
    }))
})
