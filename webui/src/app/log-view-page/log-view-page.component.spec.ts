import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { LogViewPageComponent } from './log-view-page.component'

describe('LogViewPageComponent', () => {
    let component: LogViewPageComponent
    let fixture: ComponentFixture<LogViewPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [LogViewPageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(LogViewPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
