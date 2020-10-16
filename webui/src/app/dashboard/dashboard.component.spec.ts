import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { DashboardComponent } from './dashboard.component'
import { PanelModule } from 'primeng/panel'
import { ButtonModule } from 'primeng/button'
import { Router, RouterModule, ActivatedRoute } from '@angular/router'
import { ServicesService, DHCPService, SettingsService, UsersService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { LocationStrategy } from '@angular/common'

describe('DashboardComponent', () => {
    let component: DashboardComponent
    let fixture: ComponentFixture<DashboardComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [PanelModule, ButtonModule, RouterModule, HttpClientTestingModule],
            declarations: [DashboardComponent],
            providers: [
                ServicesService,
                LocationStrategy,
                DHCPService,
                MessageService,
                UsersService,
                SettingsService,
                {
                    provide: Router,
                    useValue: {},
                },
                {
                    provide: ActivatedRoute,
                    useValue: {},
                },
            ],
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
