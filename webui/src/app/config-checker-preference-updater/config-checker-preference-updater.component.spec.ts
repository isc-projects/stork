import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ComponentFixture, TestBed } from '@angular/core/testing'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { MessageService } from 'primeng/api'
import { ChipModule } from 'primeng/chip'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TableModule } from 'primeng/table'
import { ServicesService } from '../backend'
import { ConfigCheckerPreferencePickerComponent } from '../config-checker-preference-picker/config-checker-preference-picker.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'

import { ConfigCheckerPreferenceUpdaterComponent } from './config-checker-preference-updater.component'

describe('ConfigCheckerPreferenceUpdaterComponent', () => {
    let component: ConfigCheckerPreferenceUpdaterComponent
    let fixture: ComponentFixture<ConfigCheckerPreferenceUpdaterComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [TableModule, ChipModule, OverlayPanelModule, NoopAnimationsModule, HttpClientTestingModule],
            declarations: [
                HelpTipComponent,
                ConfigCheckerPreferencePickerComponent,
                ConfigCheckerPreferenceUpdaterComponent,
            ],
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
})
