import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { SubnetsPageComponent } from './subnets-page.component'
import { FormsModule } from '@angular/forms'
import { DropdownModule } from 'primeng/dropdown'
import { TableModule } from 'primeng/table'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { TooltipModule } from 'primeng/tooltip'
import { RouterModule, ActivatedRoute, Router, convertToParamMap } from '@angular/router'
import { DHCPService, SettingsService, UsersService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { of } from 'rxjs'
import { MessageService } from 'primeng/api'

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
                UsersService,
                MessageService,
                SettingsService,
                {
                    provide: ActivatedRoute,
                    useValue: {
                        snapshot: { queryParamMap: new MockParamMap() },
                        queryParamMap: of(new MockParamMap()),
                    },
                },
                {
                    provide: Router,
                    useValue: {},
                },
            ],
            imports: [FormsModule, DropdownModule, TableModule, TooltipModule, RouterModule, HttpClientTestingModule],
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
