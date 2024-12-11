import { ComponentFixture, TestBed, fakeAsync, tick, waitForAsync } from '@angular/core/testing'
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
import { ActivatedRoute } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { DividerModule } from 'primeng/divider'
import { of, throwError } from 'rxjs'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { CheckboxModule } from 'primeng/checkbox'
import { InputNumberModule } from 'primeng/inputnumber'

describe('SettingsPageComponent', () => {
    let component: SettingsPageComponent
    let fixture: ComponentFixture<SettingsPageComponent>
    let settingsApi: SettingsService
    let messageService: MessageService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [
                BreadcrumbModule,
                BrowserAnimationsModule,
                CheckboxModule,
                DividerModule,
                FieldsetModule,
                FormsModule,
                HttpClientTestingModule,
                ReactiveFormsModule,
                MessagesModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                ProgressSpinnerModule,
                RouterTestingModule,
                InputNumberModule,
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
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(SettingsPageComponent)
        component = fixture.componentInstance
        settingsApi = fixture.debugElement.injector.get(SettingsService)
        messageService = fixture.debugElement.injector.get(MessageService)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
        expect(component.settingsForm.get('appsStatePullerInterval')?.value).toBe(0)
        expect(component.settingsForm.get('bind9StatsPullerInterval')?.value).toBe(0)
        expect(component.settingsForm.get('grafanaUrl')?.value).toBe('')
        expect(component.settingsForm.get('keaHostsPullerInterval')?.value).toBe(0)
        expect(component.settingsForm.get('keaStatsPullerInterval')?.value).toBe(0)
        expect(component.settingsForm.get('keaStatusPullerInterval')?.value).toBe(0)
        expect(component.settingsForm.get('enableMachineRegistration')?.value).toBeFalse()
        expect(component.settingsForm.get('enableOnlineSoftwareVersions')?.value).toBeFalse()
    })

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('Configuration')
        expect(breadcrumbsComponent.items[1].label).toEqual('Settings')
    })

    it('should contain the help tip', () => {
        const helptipElement = fixture.debugElement.query(By.directive(HelpTipComponent))
        expect(helptipElement).not.toBeNull()
        const helptipComponent = helptipElement.componentInstance as HelpTipComponent
        expect(helptipComponent).not.toBeNull()
        expect(helptipComponent.subject).toBe('this page')
    })

    it('should init the form', fakeAsync(() => {
        const settings: any = {
            appsStatePullerInterval: 28,
            bind9StatsPullerInterval: 29,
            grafanaUrl: 'http://localhost:1234',
            keaHostsPullerInterval: 30,
            keaStatsPullerInterval: 31,
            keaStatusPullerInterval: 32,
            enableMachineRegistration: true,
            enableOnlineSoftwareVersions: true,
        }
        spyOn(settingsApi, 'getSettings').and.returnValue(of(settings))
        component.ngOnInit()
        fixture.detectChanges()
        tick()

        expect(settingsApi.getSettings).toHaveBeenCalled()
        expect(component.settingsForm.get('appsStatePullerInterval')?.value).toBe(28)
        expect(component.settingsForm.get('bind9StatsPullerInterval')?.value).toBe(29)
        expect(component.settingsForm.get('grafanaUrl')?.value).toBe('http://localhost:1234')
        expect(component.settingsForm.get('keaHostsPullerInterval')?.value).toBe(30)
        expect(component.settingsForm.get('keaStatsPullerInterval')?.value).toBe(31)
        expect(component.settingsForm.get('keaStatusPullerInterval')?.value).toBe(32)
        expect(component.settingsForm.get('enableMachineRegistration')?.value).toBeTrue()
        expect(component.settingsForm.get('enableOnlineSoftwareVersions')?.value).toBeTrue()
    }))

    it('should display error message upon getting the settings', fakeAsync(() => {
        spyOn(settingsApi, 'getSettings').and.returnValue(throwError({ status: 404 }))
        spyOn(messageService, 'add').and.callThrough()
        component.ngOnInit()
        fixture.detectChanges()
        tick()

        // Error message should have been displayed and the retry button should be displayed.
        expect(messageService.add).toHaveBeenCalledTimes(1)
        const retryBtn = fixture.debugElement.query(By.css('[label=Retry]'))
        expect(retryBtn).not.toBeNull()

        // Simulate retrying.
        component.retry()
        tick()
        fixture.detectChanges()

        expect(messageService.add).toHaveBeenCalledTimes(2)
        expect(fixture.debugElement.nativeElement.innerText).toContain(
            'Fetching current settings from the server failed.'
        )
    }))

    it('should submit the form', fakeAsync(() => {
        const settings: any = {
            appsStatePullerInterval: 28,
            bind9StatsPullerInterval: 29,
            grafanaUrl: 'http://localhost:1234',
            keaHostsPullerInterval: 30,
            keaStatsPullerInterval: 31,
            keaStatusPullerInterval: 32,
            enableMachineRegistration: true,
            enableOnlineSoftwareVersions: true,
        }
        const updatedSettings: any = {
            appsStatePullerInterval: 13,
            bind9StatsPullerInterval: 13,
            grafanaUrl: 'http://localhost:4234',
            keaHostsPullerInterval: 13,
            keaStatsPullerInterval: 13,
            keaStatusPullerInterval: 13,
            enableMachineRegistration: false,
            enableOnlineSoftwareVersions: false,
        }
        spyOn(settingsApi, 'getSettings').and.returnValue(of(settings))
        spyOn(settingsApi, 'updateSettings').and.callThrough()
        component.ngOnInit()
        fixture.detectChanges()
        tick()

        component.settingsForm.setValue(updatedSettings)
        fixture.detectChanges()

        component.saveSettings()
        expect(settingsApi.updateSettings).toHaveBeenCalledWith(updatedSettings)
    }))

    it('should validate the form', fakeAsync(() => {
        const settings: any = {
            appsStatePullerInterval: null,
            bind9StatsPullerInterval: null,
            keaHostsPullerInterval: null,
            keaStatsPullerInterval: null,
            keaStatusPullerInterval: null,
        }
        spyOn(settingsApi, 'getSettings').and.returnValue(of(settings))
        spyOn(settingsApi, 'updateSettings').and.callThrough()
        component.ngOnInit()
        fixture.detectChanges()
        tick()

        // Iteratively correct the values.
        for (const key of Object.keys(settings)) {
            expect(component.settingsForm.invalid).toBeTrue()
            component.settingsForm.get(key)?.setValue(20)
        }
        // The form should eventually be valid.
        expect(component.settingsForm.invalid).toBeFalse()
    }))
})
