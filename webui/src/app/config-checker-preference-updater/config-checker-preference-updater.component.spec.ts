import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ComponentFixture, TestBed } from '@angular/core/testing'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { fail } from 'assert'
import { MessageService } from 'primeng/api'
import { ButtonModule } from 'primeng/button'
import { ChipModule } from 'primeng/chip'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TableModule } from 'primeng/table'
import { ToastModule } from 'primeng/toast'
import { ServicesService } from '../backend'
import { ConfigCheckerPreferencePickerComponent } from '../config-checker-preference-picker/config-checker-preference-picker.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'

import { ConfigCheckerPreferenceUpdaterComponent } from './config-checker-preference-updater.component'

describe('ConfigCheckerPreferenceUpdaterComponent', () => {
    let component: ConfigCheckerPreferenceUpdaterComponent
    let fixture: ComponentFixture<ConfigCheckerPreferenceUpdaterComponent>

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
            ],
            declarations: [HelpTipComponent, ConfigCheckerPreferenceUpdaterComponent, ConfigCheckerPreferencePickerComponent],
            providers: [MessageService, ServicesService],
        }).compileComponents()
    })

    beforeEach(() => {
        fixture = TestBed.createComponent(ConfigCheckerPreferenceUpdaterComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should fetch global preferences for an empty daemon ID', () => {
        fail('not implemented')
    })

    it('should fetch daemon preferences for an non-empty daemon ID', () => {
        fail('not implemented')
    })

    it('should handle fetching preferences errors', () => {
        fail('not implemented')
    })

    it('should set non-loading state after fetching preferences', () => {
        fail('not implemented')
    })

    it('should unsubscribe on destroy', () => {
        fail('not implemented')
    })

    it('should set loading state on submit', () => {
        fail('not implemented')
    })

    it('should update the global preferences if the daemon ID is empty', () => {
        fail('not implemented')
    })

    it('should update the daemon preferences if the daemon ID is not empty', () => {
        fail('not implemented')
    })

    it('should create toast on successful update', () => {
        fail('not implemented')
    })

    it('should create toast on failed update', () => {
        fail('not implemented')
    })

    it('should set non-loading state after update', () => {
        fail('not implemented')
    })
})
