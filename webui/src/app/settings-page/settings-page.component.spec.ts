import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { By } from '@angular/platform-browser'

import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { BrowserAnimationsModule, NoopAnimationsModule } from '@angular/platform-browser/animations'
import { FieldsetModule } from 'primeng/fieldset'
import { MessageService } from 'primeng/api'
import { HttpClientTestingModule } from '@angular/common/http/testing'

import { MessagesModule } from 'primeng/messages'

import { SettingsPageComponent } from './settings-page.component'
import { SettingsService } from '../backend/api/api'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { ActivatedRoute, Router } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'

describe('SettingsPageComponent', () => {
    let component: SettingsPageComponent
    let fixture: ComponentFixture<SettingsPageComponent>

    beforeEach(
        waitForAsync(() => {
            TestBed.configureTestingModule({
                imports: [
                    FormsModule,
                    ReactiveFormsModule,
                    BrowserAnimationsModule,
                    FieldsetModule,
                    HttpClientTestingModule,
                    MessagesModule,
                    BreadcrumbModule,
                    OverlayPanelModule,
                    NoopAnimationsModule,
                    RouterTestingModule
                ],
                declarations: [SettingsPageComponent, BreadcrumbsComponent, HelpTipComponent],
                providers: [
                    SettingsService,
                    MessageService,
                    {
                        provide: ActivatedRoute,
                        useValue: {},
                    },
                ],
            }).compileComponents()
        })
    )

    beforeEach(() => {
        fixture = TestBed.createComponent(SettingsPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('has help information about intervals configuration', () => {
        const intervalsConfigMsg = fixture.debugElement.query(By.css('#intervals-config-msg'))
        expect(intervalsConfigMsg).toBeTruthy()
    })
})
