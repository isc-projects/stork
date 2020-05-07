import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { SubnetsPageComponent } from './subnets-page.component'
import { FormsModule } from '@angular/forms'
import { DropdownModule } from 'primeng/dropdown'
import { TableModule } from 'primeng/table'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { TooltipModule } from 'primeng/tooltip'
import { RouterModule, ActivatedRoute, Router, convertToParamMap } from '@angular/router'
import { DHCPService, SettingsService } from '../backend'
import { HttpClient, HttpHandler } from '@angular/common/http'
import { of } from 'rxjs'

class MockParamMap {
    get(name: string): string | null {
        return null
    }
}

describe('SubnetsPageComponent', () => {
    let component: SubnetsPageComponent
    let fixture: ComponentFixture<SubnetsPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [
                DHCPService,
                HttpClient,
                HttpHandler,
                SettingsService,
                {
                    provide: ActivatedRoute,
                    useValue: {
                        snapshot: { queryParamMap: new MockParamMap() },
                        queryParamMap: of(new MockParamMap())
                    },
                },
                {
                    provide: Router,
                    useValue: {},
                },
            ],
            imports: [FormsModule, DropdownModule, TableModule, TooltipModule, RouterModule],
            declarations: [SubnetsPageComponent, SubnetBarComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(SubnetsPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
