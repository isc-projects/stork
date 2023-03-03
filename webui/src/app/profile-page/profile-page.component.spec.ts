import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { ProfilePageComponent } from './profile-page.component'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ServicesService, UsersService } from '../backend'
import { ActivatedRoute, Router } from '@angular/router'
import { MessageService } from 'primeng/api'
import { AuthService } from '../auth.service'
import { of } from 'rxjs'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { SettingsMenuComponent } from '../settings-menu/settings-menu.component'
import { PanelModule } from 'primeng/panel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { MenuModule } from 'primeng/menu'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { RouterTestingModule } from '@angular/router/testing'

describe('ProfilePageComponent', () => {
    let component: ProfilePageComponent
    let fixture: ComponentFixture<ProfilePageComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                MessageService,
                UsersService,
                ServicesService,
                AuthService,
                {
                    provide: ActivatedRoute,
                    useValue: {},
                },
            ],
            declarations: [ProfilePageComponent, BreadcrumbsComponent, SettingsMenuComponent, HelpTipComponent],
            imports: [
                HttpClientTestingModule,
                PanelModule,
                NoopAnimationsModule,
                BreadcrumbModule,
                MenuModule,
                OverlayPanelModule,
                NoopAnimationsModule,
                RouterTestingModule,
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(ProfilePageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(1)
        expect(breadcrumbsComponent.items[0].label).toEqual('User Profile')
    })
})
