import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { DaemonStatusComponent } from './daemon-status.component'
import { TooltipModule } from 'primeng/tooltip'
import { RouterTestingModule } from '@angular/router/testing'
import { DaemonNiceNamePipe } from '../pipes/daemon-name.pipe'
import { Tooltip } from 'chart.js'
import { provideRouter } from '@angular/router'

class Daemon {
    id = 1
    name = 'dhcp4'
}

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
