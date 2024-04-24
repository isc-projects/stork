import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { LoginScreenComponent } from './login-screen.component'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { AuthenticationMethod, GeneralService, UsersService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { RouterTestingModule } from '@angular/router/testing'
import { SelectButtonModule } from 'primeng/selectbutton'
import { ButtonModule } from 'primeng/button'
import { ActivatedRoute, Router } from '@angular/router'
import { of } from 'rxjs'
import { Version } from '@angular/core'
import { AuthService } from '../auth.service'

describe('LoginScreenComponent', () => {
    let component: LoginScreenComponent
    let fixture: ComponentFixture<LoginScreenComponent>
    let router: Router
    let route: ActivatedRoute
    let generalService: GeneralService
    let authService: AuthService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [
                ReactiveFormsModule,
                FormsModule,
                RouterTestingModule,
                HttpClientTestingModule,
                ProgressSpinnerModule,
                SelectButtonModule,
                ButtonModule,
            ],
            declarations: [LoginScreenComponent],
            providers: [GeneralService, UsersService, MessageService],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(LoginScreenComponent)
        router = fixture.debugElement.injector.get(Router)
        route = fixture.debugElement.injector.get(ActivatedRoute)
        generalService = fixture.debugElement.injector.get(GeneralService)
        authService = fixture.debugElement.injector.get(AuthService)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should extract the return URL from the query parameters', () => {
        // Arrange
        spyOnProperty(router, 'url').and.returnValue('/login')
        route.snapshot.queryParams = { returnUrl: ['dashboard'] }
        spyOn(generalService, 'getVersion').and.returnValue(of({ version: '1.0.0' } as any))
        spyOn(authService, 'getAuthenticationMethods').and.returnValue(
            of([{ name: 'name', description: 'description' } as AuthenticationMethod])
        )

        // Act
        component.ngOnInit()

        // Assert
        expect(component.returnUrl).toEqual(['/', 'dashboard'])
    })

    it('should use the root endpoint as the return URL if the query params are empty', () => {
        // Arrange
        spyOnProperty(router, 'url').and.returnValue('/login')
        route.snapshot.queryParams = {}
        spyOn(generalService, 'getVersion').and.returnValue(of({ version: '1.0.0' } as any))
        spyOn(authService, 'getAuthenticationMethods').and.returnValue(
            of([{ name: 'name', description: 'description' } as AuthenticationMethod])
        )

        // Act
        component.ngOnInit()

        // Assert
        expect(component.returnUrl).toEqual(['/'])
    })

    it('should redirect user to the previous page after login', () => {
        // Arrange
        component.returnUrl = ['/', 'dashboard']
        component.loginForm.controls = {}
        component.authenticationMethod = { id: 'foo' } as AuthenticationMethod
        component.loginForm.controls = {
            identifier: {
                value: 'bar',
                markAsDirty: () => {},
            } as any,
            secret: {
                value: 'baz',
                markAsDirty: () => {},
            } as any,
        }
        spyOnProperty(component.loginForm, 'valid').and.returnValue(true)
        const spyLogin = spyOn(authService, 'login')

        // Act
        component.signIn()

        // Assert
        expect(spyLogin).toHaveBeenCalledWith('foo', 'bar', 'baz', ['/', 'dashboard'])
    })
})
