import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { ActivatedRoute, provideRouter } from '@angular/router'

import { SettingsMenuComponent } from './settings-menu.component'
import { AuthService } from '../auth.service'
import { User } from '../backend'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

describe('SettingsMenuComponent', () => {
    let component: SettingsMenuComponent
    let fixture: ComponentFixture<SettingsMenuComponent>
    let authService: AuthService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                {
                    provide: ActivatedRoute,
                    useValue: {},
                },
                MessageService,
                provideRouter([]),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideNoopAnimations(),
            ],
        }).compileComponents()
        authService = TestBed.inject(AuthService)
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(SettingsMenuComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should show the password change menu item for internal user', () => {
        spyOnProperty(authService, 'currentUserValue', 'get').and.returnValue({
            authenticationMethodId: 'internal',
        } as User)
        component.ngOnInit()

        expect(component.items[0].items.length).toBe(2)
        expect(component.items[0].items.some((i) => i.label === 'Change password')).toBeTrue()
    })

    it('should hide the password change menu item for external user', () => {
        spyOnProperty(authService, 'currentUserValue', 'get').and.returnValue({
            authenticationMethodId: 'external',
        } as User)
        component.ngOnInit()

        expect(component.items[0].items.length).toBe(1)
        expect(component.items[0].items.some((i) => i.label === 'Change password')).toBeFalse()
    })
})
