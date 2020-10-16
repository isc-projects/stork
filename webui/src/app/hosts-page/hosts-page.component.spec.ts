import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { HostsPageComponent } from './hosts-page.component'
import { FormsModule } from '@angular/forms'
import { TableModule } from 'primeng/table'
import { DHCPService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ActivatedRoute, Router, convertToParamMap } from '@angular/router'
import { of } from 'rxjs'

class MockParamMap {
    get(name: string): string | null {
        return null
    }
}

describe('HostsPageComponent', () => {
    let component: HostsPageComponent
    let fixture: ComponentFixture<HostsPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [
                DHCPService,
                {
                    provide: ActivatedRoute,
                    useValue: {
                        snapshot: { queryParamMap: new MockParamMap() },
                        queryParamMap: of(new MockParamMap()),
                        paramMap: of(convertToParamMap({ id: 1 })),
                    },
                },
                {
                    provide: Router,
                    useValue: {},
                },
            ],
            imports: [FormsModule, TableModule, HttpClientTestingModule],
            declarations: [HostsPageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HostsPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
