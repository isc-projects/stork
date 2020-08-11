import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { AppDaemonsStatusComponent } from './app-daemons-status.component'

describe('AppDaemonsStatusComponent', () => {
    let component: AppDaemonsStatusComponent
    let fixture: ComponentFixture<AppDaemonsStatusComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [AppDaemonsStatusComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(AppDaemonsStatusComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
