import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { EntityLinkComponent } from './entity-link.component'
import { By } from '@angular/platform-browser'
import { provideRouter } from '@angular/router'

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
        expect(link.nativeElement.innerText).toEqual('[98] DHCPv4')

        // Test entity name is not displayed.
        let native = fixture.nativeElement
        expect(native.textContent).not.toContain('daemon')

        // Display entity name.
        component.showEntityName = true
        fixture.detectChanges()
        native = fixture.nativeElement
        expect(native.textContent).toContain('daemon')
    })

    it('should construct machine link', () => {
        component.entity = 'machine'
        component.attrs = { id: 5, address: '192.0.2.10' }
        component.showEntityName = false
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#machine-link'))
        expect(link.attributes.href).toEqual('/machines/5')
        expect(link.nativeElement.innerText).toEqual('[5] 192.0.2.10')

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
        expect(link.nativeElement.innerText).toEqual('[7] turtle')

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
        expect(link.nativeElement.innerText).toEqual('[7] mouse@example.org')

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
