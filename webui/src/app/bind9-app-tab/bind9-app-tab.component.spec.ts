import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { Bind9AppTabComponent } from './bind9-app-tab.component'

describe('Bind9AppTabComponent', () => {
    let component: Bind9AppTabComponent
    let fixture: ComponentFixture<Bind9AppTabComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [Bind9AppTabComponent],
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
