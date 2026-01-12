import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { DaemonStatusComponent } from './daemon-status.component'
import { provideRouter } from '@angular/router'

describe('DaemonStatusComponent', () => {
    let component: DaemonStatusComponent
    let fixture: ComponentFixture<DaemonStatusComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [provideRouter([])],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(DaemonStatusComponent)
        component = fixture.componentInstance
        component.daemon = {
            id: 1,
            name: 'dhcp4',
        }
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
