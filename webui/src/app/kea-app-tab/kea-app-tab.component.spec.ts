import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { KeaAppTabComponent } from './kea-app-tab.component'
import { RouterModule, Router, ActivatedRoute } from '@angular/router'
import { HaStatusComponent } from '../ha-status/ha-status.component'
import { TableModule } from 'primeng/table'
import { TabViewModule } from 'primeng/tabview'
import { LocaltimePipe } from '../localtime.pipe'
import { DHCPService } from '../backend'
import { HttpClient, HttpHandler } from '@angular/common/http'
import { PanelModule } from 'primeng/panel'
import { TooltipModule } from 'primeng/tooltip'
import { MessageModule } from 'primeng/message'
import { LocationStrategy } from '@angular/common'

describe('KeaAppTabComponent', () => {
    let component: KeaAppTabComponent
    let fixture: ComponentFixture<KeaAppTabComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [DHCPService, HttpClient, HttpHandler, LocationStrategy, {
                provide: ActivatedRoute,
                useValue: {}
            }, {
                provide: Router,
                useValue: {}
            }],
            imports: [RouterModule, TableModule, TabViewModule, PanelModule, TooltipModule, MessageModule],
            declarations: [KeaAppTabComponent, HaStatusComponent, LocaltimePipe],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(KeaAppTabComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
