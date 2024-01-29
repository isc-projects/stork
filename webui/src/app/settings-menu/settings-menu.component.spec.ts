import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { ActivatedRoute } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { MenuModule } from 'primeng/menu'

import { SettingsMenuComponent } from './settings-menu.component'
import { AuthService } from '../auth.service'
import { User } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'

describe('SettingsMenuComponent', () => {
    let component: SettingsMenuComponent
    let fixture: ComponentFixture<SettingsMenuComponent>
    let authService: AuthService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [MenuModule, NoopAnimationsModule, RouterTestingModule, HttpClientTestingModule],
            declarations: [SettingsMenuComponent],
            providers: [
                {
                    provide: ActivatedRoute,
                    useValue: {},
                },
                MessageService,
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
