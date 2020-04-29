import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { SharedNetworksPageComponent } from './shared-networks-page.component'
import { FormsModule } from '@angular/forms'
import { DropdownModule } from 'primeng/dropdown'
import { TableModule } from 'primeng/table'
import { TooltipModule } from 'primeng/tooltip'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { Router, ActivatedRoute } from '@angular/router'
import { DHCPService } from '../backend'
import { HttpClient, HttpHandler } from '@angular/common/http'

describe('SharedNetworksPageComponent', () => {
    let component: SharedNetworksPageComponent
    let fixture: ComponentFixture<SharedNetworksPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [FormsModule, DropdownModule, TableModule, TooltipModule],
            declarations: [SharedNetworksPageComponent, SubnetBarComponent],
            providers: [{
                provide: Router, useValue: {}
            },{
                provide: ActivatedRoute, useValue: {}
            }, DHCPService, HttpClient, HttpHandler]
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(SharedNetworksPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
