import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { DashboardComponent } from './dashboard.component'
import { PanelModule } from 'primeng/panel'
import { ButtonModule } from 'primeng/button'
import { Router, RouterModule, ActivatedRoute } from '@angular/router'
import { ServicesService, DHCPService, SettingsService } from '../backend'
import { HttpClient, HttpHandler } from '@angular/common/http'
import { MessageService } from 'primeng/api'
import { LocationStrategy } from '@angular/common'

describe('DashboardComponent', () => {
    let component: DashboardComponent
    let fixture: ComponentFixture<DashboardComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [PanelModule, ButtonModule, RouterModule],
            declarations: [DashboardComponent],
            providers: [ ServicesService, HttpClient, HttpHandler, LocationStrategy, DHCPService, MessageService, SettingsService, {
                provide: Router,
                useValue: {}
            },{
                provide: ActivatedRoute,
                useValue: {}
            }]
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(DashboardComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
