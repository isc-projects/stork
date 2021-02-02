import { async, ComponentFixture, TestBed } from '@angular/core/testing'
import { EntityLinkComponent } from './entity-link.component'
import { RouterTestingModule } from '@angular/router/testing'
import { By } from '@angular/platform-browser'

describe('EntityLinkComponent', () => {
    let component: EntityLinkComponent
    let fixture: ComponentFixture<EntityLinkComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [RouterTestingModule],
            declarations: [EntityLinkComponent],
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
        component.attrs = { id: 98, appType: 'kea', appId: 1, name: 'dhcp4' }
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#daemon-link'))
        expect(link.attributes.href).toEqual('/apps/kea/1?daemon=dhcp4')
        expect(link.nativeElement.innerText).toEqual('[98] dhcp4')
    })

    it('should construct app link', () => {
        component.entity = 'app'
        component.attrs = { type: 'kea', id: 1, name: 'test-app' }
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#app-link'))
        expect(link.attributes.href).toEqual('/apps/kea/1')
        expect(link.nativeElement.innerText).toEqual('test-app')
    })

    it('should construct machine link', () => {
        component.entity = 'machine'
        component.attrs = { id: 5, address: '192.0.2.10' }
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#machine-link'))
        expect(link.attributes.href).toEqual('/machines/5')
        expect(link.nativeElement.innerText).toEqual('[5] 192.0.2.10')
    })

    it('should construct user link with login', () => {
        component.entity = 'user'
        component.attrs = { id: 7, login: 'user1' }
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#user-link'))
        expect(link.attributes.href).toEqual('/users/7')
        expect(link.nativeElement.innerText).toEqual('[7] user1')
    })

    it('should construct user link with email', () => {
        component.entity = 'user'
        component.attrs = { id: 7, email: 'user1@example.org' }
        fixture.detectChanges()
        const link = fixture.debugElement.query(By.css('#user-link'))
        expect(link.attributes.href).toEqual('/users/7')
        expect(link.nativeElement.innerText).toEqual('[7] user1@example.org')
    })
})
