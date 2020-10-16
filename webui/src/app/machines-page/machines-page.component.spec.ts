import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { MachinesPageComponent } from './machines-page.component'
import { ActivatedRoute, Router, convertToParamMap } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { ServicesService, UsersService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { of } from 'rxjs'

describe('MachinesPageComponent', () => {
    let component: MachinesPageComponent
    let fixture: ComponentFixture<MachinesPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [MessageService, ServicesService, UsersService],
            imports: [
                HttpClientTestingModule,
                RouterTestingModule.withRoutes([{ path: 'machines/all', component: MachinesPageComponent }]),
            ],
            declarations: [MachinesPageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(MachinesPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
