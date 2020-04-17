import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { HaStatusComponent } from './ha-status.component'
import { PanelModule } from 'primeng/panel'
import { TooltipModule } from 'primeng/tooltip'
import { MessageModule } from 'primeng/message'
import { LocaltimePipe } from '../localtime.pipe'
import { RouterModule } from '@angular/router'
import { ServicesService } from '../backend'
import { HttpClient, HttpHandler } from '@angular/common/http'

describe('HaStatusComponent', () => {
    let component: HaStatusComponent
    let fixture: ComponentFixture<HaStatusComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [PanelModule, TooltipModule, MessageModule, RouterModule],
            declarations: [HaStatusComponent, LocaltimePipe],
            providers: [ServicesService, HttpClient, HttpHandler]
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HaStatusComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
