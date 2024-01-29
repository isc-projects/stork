import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { PasswordChangePageComponent } from './password-change-page.component'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ActivatedRoute } from '@angular/router'
import { UntypedFormBuilder, ReactiveFormsModule } from '@angular/forms'
import { UsersService } from '../backend'
import { MessageService } from 'primeng/api'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { PanelModule } from 'primeng/panel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { SettingsMenuComponent } from '../settings-menu/settings-menu.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { MenuModule } from 'primeng/menu'
import { RouterTestingModule } from '@angular/router/testing'
import { PasswordModule } from 'primeng/password'
import { MessageModule } from 'primeng/message'

describe('PasswordChangePageComponent', () => {
    let component: PasswordChangePageComponent
    let fixture: ComponentFixture<PasswordChangePageComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                UntypedFormBuilder,
                UsersService,
                MessageService,
                {
                    provide: ActivatedRoute,
                    useValue: {},
                },
            ],
            imports: [
                HttpClientTestingModule,
                PanelModule,
                NoopAnimationsModule,
                BreadcrumbModule,
                OverlayPanelModule,
                MenuModule,
                RouterTestingModule,
                ReactiveFormsModule,
                PasswordModule,
                MessageModule,
            ],
            declarations: [PasswordChangePageComponent, BreadcrumbsComponent, SettingsMenuComponent, HelpTipComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(PasswordChangePageComponent)
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
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('User Profile')
        expect(breadcrumbsComponent.items[1].label).toEqual('Password Change')
    })
})
