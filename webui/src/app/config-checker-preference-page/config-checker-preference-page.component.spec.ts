import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { RouterTestingModule } from '@angular/router/testing'
import { MessageService } from 'primeng/api'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { ButtonModule } from 'primeng/button'
import { ChipModule } from 'primeng/chip'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TableModule } from 'primeng/table'
import { ToastModule } from 'primeng/toast'
import { ServicesService } from '../backend'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { ConfigCheckerPreferencePickerComponent } from '../config-checker-preference-picker/config-checker-preference-picker.component'
import { ConfigCheckerPreferenceUpdaterComponent } from '../config-checker-preference-updater/config-checker-preference-updater.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'

import { ConfigCheckerPreferencePageComponent } from './config-checker-preference-page.component'

describe('ConfigCheckerPreferencePageComponent', () => {
    let component: ConfigCheckerPreferencePageComponent
    let fixture: ComponentFixture<ConfigCheckerPreferencePageComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [
                TableModule,
                ChipModule,
                OverlayPanelModule,
                NoopAnimationsModule,
                HttpClientTestingModule,
                ToastModule,
                BreadcrumbModule,
                RouterTestingModule,
                ButtonModule,
            ],
            declarations: [
                HelpTipComponent,
                BreadcrumbsComponent,
                ConfigCheckerPreferencePageComponent,
                ConfigCheckerPreferenceUpdaterComponent,
                ConfigCheckerPreferencePickerComponent,
            ],
            providers: [MessageService, ServicesService],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(ConfigCheckerPreferencePageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display the config review checkers panel with the full layout', () => {
        const element = fixture.debugElement.query(By.directive(ConfigCheckerPreferencePickerComponent))
        expect(element).not.toBeNull()
        const picker = element.componentInstance as ConfigCheckerPreferencePickerComponent

        expect(picker.allowInheritState).toBeFalse()
        expect(picker.minimal).toBeFalse()
    })

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('Configuration')
        expect(breadcrumbsComponent.items[1].label).toEqual('Review Checkers')
    })
})
