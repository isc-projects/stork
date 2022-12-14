import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { HaStatusComponent } from './ha-status.component'
import { PanelModule } from 'primeng/panel'
import { TooltipModule } from 'primeng/tooltip'
import { MessageModule } from 'primeng/message'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { RouterModule } from '@angular/router'
import { ServicesService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'

describe('HaStatusComponent', () => {
    let component: HaStatusComponent
    let fixture: ComponentFixture<HaStatusComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [PanelModule, TooltipModule, MessageModule, RouterModule, HttpClientTestingModule],
            declarations: [HaStatusComponent, LocaltimePipe],
            providers: [ServicesService],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HaStatusComponent)
        component = fixture.componentInstance
        component.appId = 4
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
