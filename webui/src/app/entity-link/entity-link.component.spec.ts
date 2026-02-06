import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { EntityLinkComponent } from './entity-link.component'
import { By } from '@angular/platform-browser'
import { provideRouter } from '@angular/router'
import { Daemon, LeasesSearchErredDaemon, LocalHost, LocalSharedNetwork, LocalSubnet, LocalZone } from '../backend'

describe('EntityLinkComponent', () => {
    let component: EntityLinkComponent
    let fixture: ComponentFixture<EntityLinkComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [provideRouter([])],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(EntityLinkComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should construct daemon link', () => {
        component.entity = 'daemon'
        component.attrs = { id: 98, name: 'dhcp4' }
        component.showEntityName = false
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#daemon-link-98'))
        expect(link.attributes.href).toEqual('/daemons/98')
        expect(link.nativeElement.innerText).toEqual('[98]\u00a0DHCPv4')

        // Test entity name is not displayed.
        let native = fixture.nativeElement
        expect(native.textContent).not.toContain('daemon')

        // Display entity name.
        component.showEntityName = true
        fixture.detectChanges()
        native = fixture.nativeElement
        expect(native.textContent).toContain('daemon')
        component.showEntityName = false

        // Test that entity link from LocalSubnet.
        const subnet: LocalSubnet = {
            id: 24,
            daemonId: 42,
            daemonLabel: 'DHCPv4@localhost',
        }
        component.attrs = subnet
        fixture.detectChanges()
        expect(link.nativeElement.innerText).toEqual('[42]\u00a0DHCPv4@localhost')

        // Test entity link from LocalSharedNetwork.
        const sharedNetwork: LocalSharedNetwork = {
            daemonId: 42,
            daemonLabel: 'DHCPv4@localhost',
        }
        component.attrs = sharedNetwork
        fixture.detectChanges()
        expect(link.nativeElement.innerText).toEqual('[42]\u00a0DHCPv4@localhost')

        // Test entity link from LocalHost.
        const host: LocalHost = {
            bootFileName: 'pxelinux.0',
            clientClasses: ['class1', 'class2'],
            dataSource: 'api',
            hostname: 'host',
            serverHostname: 'server',
            daemonId: 42,
            daemonLabel: 'DHCPv4@localhost',
        }
        component.attrs = host
        fixture.detectChanges()
        expect(link.nativeElement.innerText).toEqual('[42]\u00a0DHCPv4@localhost')

        // Test entity link from LocalZone.
        const zone: LocalZone = {
            daemonId: 42,
            daemonLabel: 'DHCPv4@localhost',
            loadedAt: '2024-06-01T12:00:00Z',
            rpz: true,
            serial: 12345,
            view: 'default',
            zoneClass: 'IN',
            zoneType: 'primary',
        }
        component.attrs = zone
        fixture.detectChanges()
        expect(link.nativeElement.innerText).toEqual('[42]\u00a0DHCPv4@localhost')

        // Test entity link from Daemon.
        const daemon: Daemon = {
            id: 42,
            name: 'dhcp4',
            label: 'DHCPv4@localhost',
            active: true,
            machineId: 1,
            machineLabel: 'localhost',
            monitored: true,
            pid: 1234,
            version: '1.0.0',
        }
        component.attrs = daemon
        fixture.detectChanges()
        expect(link.nativeElement.innerText).toEqual('[42]\u00a0DHCPv4')

        // Test entity link from LeasesSearchErredDaemon.
        const erredDaemon: LeasesSearchErredDaemon = {
            id: 42,
            label: 'DHCPv4@localhost',
        }
        component.attrs = erredDaemon
        fixture.detectChanges()
        expect(link.nativeElement.innerText).toEqual('[42]\u00a0DHCPv4@localhost')
    })

    it('should construct machine link', () => {
        component.entity = 'machine'
        component.attrs = { id: 5, address: '192.0.2.10' }
        component.showEntityName = false
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#machine-link'))
        expect(link.attributes.href).toEqual('/machines/5')
        expect(link.nativeElement.innerText).toEqual('[5]\u00a0192.0.2.10')

        // Test entity name is not displayed.
        let native = fixture.nativeElement
        expect(native.textContent).not.toContain('machine')

        // Display entity name.
        component.showEntityName = true
        fixture.detectChanges()
        native = fixture.nativeElement
        expect(native.textContent).toContain('machine')
    })

    it('should construct user link with login', () => {
        component.entity = 'user'
        component.attrs = { id: 7, login: 'turtle' }
        component.showEntityName = false
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#user-link'))
        expect(link.attributes.href).toEqual('/users/7')
        expect(link.nativeElement.innerText).toEqual('[7]\u00a0turtle')

        // Test entity name is not displayed.
        let native = fixture.nativeElement
        expect(native.textContent).not.toContain('user')

        // Display entity name.
        component.showEntityName = true
        fixture.detectChanges()
        native = fixture.nativeElement
        expect(native.textContent).toContain('user')
    })

    it('should construct user link with email', () => {
        component.entity = 'user'
        component.attrs = { id: 7, email: 'mouse@example.org' }
        component.showEntityName = false
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#user-link'))
        expect(link.attributes.href).toEqual('/users/7')
        expect(link.nativeElement.innerText).toEqual('[7]\u00a0mouse@example.org')

        // Test entity name is not displayed.
        let native = fixture.nativeElement
        expect(native.textContent).not.toContain('user')

        // Display entity name.
        component.showEntityName = true
        fixture.detectChanges()
        native = fixture.nativeElement
        expect(native.textContent).toContain('user')
    })

    it('should construct host link', () => {
        component.entity = 'host'
        component.attrs = { id: 8, label: 'mouse.example.org' }
        component.showEntityName = false
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#host-link'))
        expect(link.attributes.href).toEqual('/dhcp/hosts/8')
        expect(link.nativeElement.innerText).toEqual('mouse.example.org')

        // Test entity name is not displayed.
        let native = fixture.nativeElement
        expect(native.textContent).not.toContain('host')

        // Display entity name.
        component.showEntityName = true
        fixture.detectChanges()
        native = fixture.nativeElement
        expect(native.textContent).toContain('host')
    })

    it('should construct subnet link', () => {
        component.entity = 'subnet'
        component.attrs = { id: 8, subnet: 'fe80::/64' }
        component.showEntityName = false
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#subnet-link'))
        expect(link.attributes.href).toEqual('/dhcp/subnets/8')
        expect(link.nativeElement.innerText).toEqual('fe80::/64')

        // Test entity name is not displayed.
        let native = fixture.nativeElement
        expect(native.textContent).not.toContain('subnet')

        // Display entity name.
        component.showEntityName = true
        fixture.detectChanges()
        native = fixture.nativeElement
        expect(native.textContent).toContain('subnet')
    })

    it('should construct a shared network link', () => {
        component.entity = 'shared-network'
        component.attrs = { id: 9, name: 'frog' }
        component.showEntityName = false
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#shared-network-link'))
        expect(link.attributes.href).toEqual('/dhcp/shared-networks/9')
        expect(link.nativeElement.innerText).toEqual('frog')

        // Test entity name is not displayed.
        let native = fixture.nativeElement
        expect(native.textContent).not.toContain('subnet')

        // Display entity name.
        component.showEntityName = true
        fixture.detectChanges()
        native = fixture.nativeElement
        expect(native.textContent).toContain('shared network')
    })
})
