import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { HostsPageComponent } from './hosts-page.component'
import { FormsModule } from '@angular/forms'
import { TableModule } from 'primeng/table'
import { DHCPService } from '../backend'
import { HttpClient, HttpHandler } from '@angular/common/http'
import { ActivatedRoute, Router } from '@angular/router'
import { of } from 'rxjs'

describe('HostsPageComponent', () => {
    let component: HostsPageComponent
    let fixture: ComponentFixture<HostsPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [
                DHCPService,
                HttpClient,
                HttpHandler,
                {
                    provide: ActivatedRoute,
                    useValue: {
                        snapshot: { queryParams: {} },
                        queryParamMap: of({}),
                    },
                },
                {
                    provide: Router,
                    useValue: {},
                },
            ],
            imports: [FormsModule, TableModule],
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
