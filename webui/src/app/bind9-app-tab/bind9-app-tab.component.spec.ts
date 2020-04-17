import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { Bind9AppTabComponent } from './bind9-app-tab.component'
import { RouterLink, Router, RouterModule, ActivatedRoute } from '@angular/router'
import { TooltipModule } from 'primeng/tooltip'
import { TabViewModule } from 'primeng/tabview'
import { LocaltimePipe } from '../localtime.pipe'
import { LocationStrategy } from '@angular/common'

describe('Bind9AppTabComponent', () => {
    let component: Bind9AppTabComponent
    let fixture: ComponentFixture<Bind9AppTabComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [TooltipModule, TabViewModule, RouterModule],
            declarations: [Bind9AppTabComponent, LocaltimePipe],
            providers: [ LocationStrategy, {
                provide: Router,
                useValue: {}
            }, {
                provide: ActivatedRoute,
                useValue: {}

            }]
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(Bind9AppTabComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
