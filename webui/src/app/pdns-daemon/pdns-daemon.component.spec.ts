import { ComponentFixture, TestBed } from '@angular/core/testing'

import { PdnsDaemonComponent } from './pdns-daemon.component'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { DurationPipe } from '../pipes/duration.pipe'
import { PdnsDaemon } from '../backend'

const daemon: PdnsDaemon = {
    name: 'pdns',
    id: 1,
    pid: 1,
    active: true,
    monitored: true,
    version: '4.1.2',
    uptime: 100,
    url: 'http://localhost:5380',
    configUrl: 'http://localhost:5380/config',
    zonesUrl: 'http://localhost:5380/zones',
    autoprimariesUrl: 'http://localhost:5380/autoprimaries',
}

describe('PdnsDaemonComponent', () => {
    let component: PdnsDaemonComponent
    let fixture: ComponentFixture<PdnsDaemonComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [PdnsDaemonComponent, DurationPipe, PlaceholderPipe],
        }).compileComponents()

        fixture = TestBed.createComponent(PdnsDaemonComponent)
        component = fixture.componentInstance
        component.daemon = daemon
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
