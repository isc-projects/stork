import { HttpEvent } from '@angular/common/http'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ComponentFixture, fakeAsync, TestBed, tick } from '@angular/core/testing'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { MessageService } from 'primeng/api'
import { ButtonModule } from 'primeng/button'
import { ChipModule } from 'primeng/chip'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TableModule } from 'primeng/table'
import { ToastModule } from 'primeng/toast'
import { of, throwError } from 'rxjs'
import { ConfigChecker, ConfigCheckers, ServicesService } from '../backend'
import { ConfigCheckerPreferencePickerComponent } from '../config-checker-preference-picker/config-checker-preference-picker.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'

import { ConfigCheckerPreferenceUpdaterComponent } from './config-checker-preference-updater.component'
import { TriStateCheckboxModule } from 'primeng/tristatecheckbox'
import { FormsModule } from '@angular/forms'
import { TagModule } from 'primeng/tag'

describe('ConfigCheckerPreferenceUpdaterComponent', () => {
    let component: ConfigCheckerPreferenceUpdaterComponent
    let fixture: ComponentFixture<ConfigCheckerPreferenceUpdaterComponent>
    let servicesApi: ServicesService
    let messageService: MessageService
    let messageAddSpy: jasmine.Spy

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [
                TableModule,
                ChipModule,
                OverlayPanelModule,
                NoopAnimationsModule,
                HttpClientTestingModule,
                ToastModule,
                ButtonModule,
                FormsModule,
                TriStateCheckboxModule,
                TagModule,
            ],
            declarations: [
                HelpTipComponent,
                ConfigCheckerPreferenceUpdaterComponent,
                ConfigCheckerPreferencePickerComponent,
            ],
            providers: [MessageService, ServicesService],
        }).compileComponents()

        servicesApi = TestBed.inject(ServicesService)
    })

    beforeEach(() => {
        spyOn(servicesApi, 'getGlobalConfigCheckers').and.returnValue(
            of({
                total: 2,
                items: [
                    {
                        name: 'out_of_pool_reservation',
                        selectors: ['each-daemon', 'kea-daemon'],
                        state: ConfigChecker.StateEnum.Disabled,
                        triggers: ['manual', 'config change'],
                        globallyEnabled: false,
                    },
                    {
                        name: 'dispensable_subnet',
                        selectors: ['each-daemon'],
                        state: ConfigChecker.StateEnum.Enabled,
                        triggers: ['manual', 'config change'],
                        globallyEnabled: true,
                    },
                ],
            } as ConfigCheckers & HttpEvent<ConfigCheckers>)
        )

        spyOn(servicesApi, 'getDaemonConfigCheckers').and.returnValue(
            throwError({
                ok: false,
                status: 500,
                statusText: 'Error',
            } as HttpEvent<ConfigCheckers>)
        )

        spyOn(servicesApi, 'putGlobalConfigCheckerPreferences').and.returnValue(
            throwError({
                ok: false,
                status: 500,
                statusText: 'Error',
            } as HttpEvent<ConfigCheckers>)
        )

        spyOn(servicesApi, 'putDaemonConfigCheckerPreferences').and.returnValue(
            of({
                total: 0,
                items: [],
            } as ConfigCheckers & HttpEvent<ConfigCheckers>)
        )

        fixture = TestBed.createComponent(ConfigCheckerPreferenceUpdaterComponent)
        component = fixture.componentInstance
        messageService = fixture.debugElement.injector.get(MessageService)

        messageAddSpy = spyOn(messageService, 'add')
    })

    it('should create', () => {
        fixture.detectChanges()
        expect(component).toBeTruthy()
    })

    it('should fetch global preferences for an empty daemon ID', () => {
        component.daemonID = null
        fixture.detectChanges()
        expect(servicesApi.getGlobalConfigCheckers).toHaveBeenCalled()
        expect(servicesApi.getDaemonConfigCheckers).not.toHaveBeenCalled()
    })

    it('should fetch daemon preferences for an non-empty daemon ID', () => {
        component.daemonID = 42
        fixture.detectChanges()
        expect(servicesApi.getGlobalConfigCheckers).not.toHaveBeenCalled()
        expect(servicesApi.getDaemonConfigCheckers).toHaveBeenCalled()
    })

    it('should handle fetching preferences errors', () => {
        component.daemonID = 42
        fixture.detectChanges()
        expect(component.checkers).not.toBeNull()
        expect(component.checkers).not.toBeUndefined()
        expect(component.checkers.length).toBe(0)
        expect(component.loading).toBeFalse()
    })

    it('should set non-loading state after fetching preferences', () => {
        expect(component.loading).toBeTrue()
        fixture.detectChanges()
        expect(component.loading).toBeFalse()
    })

    it('should set loading state on submit', fakeAsync(() => {
        fixture.detectChanges()
        expect(component.loading).toBeFalse()
        component.onChangePreferences([
            {
                name: 'foo',
                state: 'disabled',
            },
        ])
        expect(component.loading).toBeTrue()
        tick()
        expect(component.loading).toBeFalse()
    }))

    it('should update the global preferences if the daemon ID is empty', fakeAsync(() => {
        fixture.detectChanges()
        component.onChangePreferences([
            {
                name: 'foo',
                state: 'disabled',
            },
        ])
        tick()
        fixture.detectChanges()
        expect(servicesApi.putGlobalConfigCheckerPreferences).toHaveBeenCalled()
        expect(servicesApi.putDaemonConfigCheckerPreferences).not.toHaveBeenCalled()
        expect(component.loading).toBeFalse()
    }))

    it('should update the daemon preferences if the daemon ID is not empty', fakeAsync(() => {
        component.daemonID = 42
        fixture.detectChanges()
        component.onChangePreferences([
            {
                name: 'foo',
                state: 'disabled',
            },
        ])
        tick()
        fixture.detectChanges()
        expect(servicesApi.putGlobalConfigCheckerPreferences).not.toHaveBeenCalled()
        expect(servicesApi.putDaemonConfigCheckerPreferences).toHaveBeenCalled()
        expect(component.loading).toBeFalse()
    }))

    it('should create message on successful update', fakeAsync(() => {
        component.daemonID = 42
        fixture.detectChanges()
        messageAddSpy.calls.reset()

        component.onChangePreferences([
            {
                name: 'foo',
                state: 'disabled',
            },
        ])
        tick()
        fixture.detectChanges()
        expect(messageService.add).toHaveBeenCalledOnceWith(
            jasmine.objectContaining({
                severity: 'success',
            })
        )
    }))

    it('should create message on failed update', fakeAsync(() => {
        fixture.detectChanges()
        messageAddSpy.calls.reset()

        component.onChangePreferences([
            {
                name: 'foo',
                state: 'disabled',
            },
        ])
        tick()
        fixture.detectChanges()
        expect(messageService.add).toHaveBeenCalledOnceWith(
            jasmine.objectContaining({
                severity: 'error',
            })
        )
    }))
})
