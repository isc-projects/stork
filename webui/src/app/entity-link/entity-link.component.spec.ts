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
    })

    /** Asserts that the daemon link with the given id has the expected text and href. */
    function expectDaemonLink(linkId: number | string, text: string, href: string): void {
        const link = fixture.debugElement.query(By.css(`#daemon-link-${linkId}`))
        expect(link.nativeElement.innerText).toEqual(text)
        expect(link.attributes.href).toEqual(href)
    }

    /** Renders the component as a daemon link with the given attributes. */
    function renderDaemon(attrs: unknown): void {
        component.entity = 'daemon'
        component.attrs = attrs
        component.showEntityName = false
        fixture.detectChanges()
    }

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should construct daemon link', () => {
        renderDaemon({ id: 98, name: 'dhcp4' })
        expectDaemonLink(98, '[98]\u00a0DHCPv4', '/daemons/98')
        expect(fixture.nativeElement.textContent).not.toContain('daemon')
    })

    it('should construct daemon link from LocalSubnet', () => {
        const subnet: LocalSubnet = {
            id: 24,
            daemonId: 42,
            daemonLabel: 'DHCPv4@localhost',
        }
        renderDaemon(subnet)
        expectDaemonLink(42, '[42]\u00a0DHCPv4@localhost', '/daemons/42')
    })

    it('should construct daemon link from LocalSharedNetwork', () => {
        const sharedNetwork: LocalSharedNetwork = {
            daemonId: 42,
            daemonLabel: 'DHCPv4@localhost',
        }
        renderDaemon(sharedNetwork)
        expectDaemonLink(42, '[42]\u00a0DHCPv4@localhost', '/daemons/42')
    })

    it('should construct daemon link from LocalHost', () => {
        const host: LocalHost = {
            bootFileName: 'pxelinux.0',
            clientClasses: ['class1', 'class2'],
            dataSource: 'api',
            hostname: 'host',
            serverHostname: 'server',
            daemonId: 42,
            daemonLabel: 'DHCPv4@localhost',
        }
        renderDaemon(host)
        expectDaemonLink(42, '[42]\u00a0DHCPv4@localhost', '/daemons/42')
    })

    it('should construct daemon link from LocalZone', () => {
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
        renderDaemon(zone)
        expectDaemonLink(42, '[42]\u00a0DHCPv4@localhost', '/daemons/42')
    })

    it('should construct daemon link from Daemon', () => {
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
        renderDaemon(daemon)
        expectDaemonLink(42, '[42]\u00a0DHCPv4', '/daemons/42')
    })

    it('should construct daemon link from LeasesSearchErredDaemon', () => {
        const erredDaemon: LeasesSearchErredDaemon = {
            id: 42,
            label: 'DHCPv4@localhost',
        }
        renderDaemon(erredDaemon)
        expectDaemonLink(42, '[42]\u00a0DHCPv4@localhost', '/daemons/42')
    })

    it('should construct daemon link without id', () => {
        renderDaemon({ name: 'dhcp4' })
        expectDaemonLink(0, 'DHCPv4', '/daemons')
    })

    it('should construct daemon link for numeric zero id', () => {
        renderDaemon({ id: 0, name: 'dhcp4' })
        expectDaemonLink(0, 'DHCPv4', '/daemons')
    })

    it('should construct daemon link for string zero id', () => {
        renderDaemon({ id: '0', name: 'dhcp4' })
        expectDaemonLink(0, 'DHCPv4', '/daemons')
    })

    it('should construct machine link', () => {
        component.entity = 'machine'
        component.attrs = { id: 5, address: '192.0.2.10' }
        component.showEntityName = false
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#machine-link'))
        expect(link.attributes.href).toEqual('/machines/5')
        expect(link.nativeElement.innerText).toEqual('[5]\u00a0192.0.2.10')
        expect(fixture.nativeElement.textContent).not.toContain('machine')
    })

    it('should construct user link with login', () => {
        component.entity = 'user'
        component.attrs = { id: 7, login: 'turtle' }
        component.showEntityName = false
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#user-link'))
        expect(link.attributes.href).toEqual('/users/7')
        expect(link.nativeElement.innerText).toEqual('[7]\u00a0turtle')
        expect(fixture.nativeElement.textContent).not.toContain('user')
    })

    it('should construct user link with email', () => {
        component.entity = 'user'
        component.attrs = { id: 7, email: 'mouse@example.org' }
        component.showEntityName = false
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#user-link'))
        expect(link.attributes.href).toEqual('/users/7')
        expect(link.nativeElement.innerText).toEqual('[7]\u00a0mouse@example.org')
        expect(fixture.nativeElement.textContent).not.toContain('user')
    })

    it('should construct host link', () => {
        component.entity = 'host'
        component.attrs = { id: 8, label: 'mouse.example.org' }
        component.showEntityName = false
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#host-link'))
        expect(link.attributes.href).toEqual('/dhcp/hosts/8')
        expect(link.nativeElement.innerText).toEqual('mouse.example.org')
        expect(fixture.nativeElement.textContent).not.toContain('host')
    })

    it('should show host identifier when requested', () => {
        component.entity = 'host'
        component.attrs = { id: 8, label: 'mouse.example.org' }
        component.showEntityName = false
        fixture.componentRef.setInput('showIdentifier', true)
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#host-link'))
        expect(link.nativeElement.innerText).toEqual('[8]\u00a0mouse.example.org')
    })

    it('should construct subnet link', () => {
        component.entity = 'subnet'
        component.attrs = { id: 8, subnet: 'fe80::/64' }
        component.showEntityName = false
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#subnet-link'))
        expect(link.attributes.href).toEqual('/dhcp/subnets/8')
        expect(link.nativeElement.innerText).toEqual('fe80::/64')
        expect(fixture.nativeElement.textContent).not.toContain('subnet')
    })

    it('should show subnet identifier when requested', () => {
        component.entity = 'subnet'
        component.attrs = { id: 8, subnet: 'fe80::/64' }
        component.showEntityName = false
        fixture.componentRef.setInput('showIdentifier', true)
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#subnet-link'))
        expect(link.nativeElement.innerText).toEqual('[8] fe80::/64')
    })

    it('should construct a shared network link', () => {
        component.entity = 'shared-network'
        component.attrs = { id: 9, name: 'frog' }
        component.showEntityName = false
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#shared-network-link'))
        expect(link.attributes.href).toEqual('/dhcp/shared-networks/9')
        expect(link.nativeElement.innerText).toEqual('frog')
        expect(fixture.nativeElement.textContent).not.toContain('subnet')
    })

    it('should show shared network identifier when requested', () => {
        component.entity = 'shared-network'
        component.attrs = { id: 9, name: 'frog' }
        component.showEntityName = false
        fixture.componentRef.setInput('showIdentifier', true)
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#shared-network-link'))
        expect(link.nativeElement.innerText).toEqual('[9]\u00a0frog')
    })
})
