import { ComponentFixture, TestBed } from '@angular/core/testing'

import { PdnsDaemonComponent } from './pdns-daemon.component'
import { PdnsDaemon } from '../backend'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { ConfirmationService, MessageService } from 'primeng/api'

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
            providers: [
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                MessageService,
                ConfirmationService,
            ],
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
