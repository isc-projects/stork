import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing'

import { AccessPointKeyComponent } from './access-point-key.component'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ButtonModule } from 'primeng/button'
import { ServicesService } from '../backend'
import { of, throwError } from 'rxjs'
import { HttpEvent } from '@angular/common/http'

describe('AccessPointKeyComponent', () => {
    let component: AccessPointKeyComponent
    let fixture: ComponentFixture<AccessPointKeyComponent>
    let servicesApi: ServicesService

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [HttpClientTestingModule, ButtonModule],
            providers: [ServicesService],
            declarations: [AccessPointKeyComponent],
        }).compileComponents()

        servicesApi = TestBed.inject(ServicesService)

        fixture = TestBed.createComponent(AccessPointKeyComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should recognize whether the key is loading or not', () => {
        // Key is not loading and not fetched.
        component.key = undefined
        expect(component.isKeyLoading).toBeFalse()

        // Key is loading.
        component.key = null
        expect(component.isKeyLoading).toBeTrue()

        // Key is fetched.
        component.key = 'foo'
        expect(component.isKeyLoading).toBeFalse()

        // Key is fetched but empty.
        component.key = ''
        expect(component.isKeyLoading).toBeFalse()
    })

    it('should recognize whether the key is fetched or not', () => {
        // Key is not fetched.
        component.key = undefined
        expect(component.isKeyFetched).toBeFalse()

        // Key is loading.
        component.key = null
        expect(component.isKeyFetched).toBeFalse()

        // Key is fetched.
        component.key = 'foo'
        expect(component.isKeyFetched).toBeTrue()

        // Key is fetched but empty.
        component.key = ''
        expect(component.isKeyFetched).toBeTrue()
    })

    it('should recognize if the key is fetched and empty', () => {
        // Key is not fetched.
        component.key = undefined
        expect(component.isKeyEmpty).toBeFalse()

        // Key is loading.
        component.key = null
        expect(component.isKeyEmpty).toBeFalse()

        // Key is fetched.
        component.key = 'foo'
        expect(component.isKeyEmpty).toBeFalse()

        // Key is fetched but empty.
        component.key = ''
        expect(component.isKeyEmpty).toBeTrue()
    })

    it('should set loading state during the key fetching', fakeAsync(() => {
        component.appId = 42
        component.accessPointType = 'control'
        spyOn(servicesApi, 'getAccessPointKey').and.returnValue(of('foobar' as string & HttpEvent<string>))
        component.onAuthenticationKeyRequest()
        expect(component.isKeyLoading).toBeTrue()
    }))

    it('should fetch key from API', fakeAsync(() => {
        component.appId = 42
        component.accessPointType = 'control'
        spyOn(servicesApi, 'getAccessPointKey').and.returnValue(of('foobar' as string & HttpEvent<string>))
        component.onAuthenticationKeyRequest()
        tick()
        expect(component.key).toBe('foobar')
        expect(component.isKeyFetched).toBeTrue()
    }))

    it('should reset the key on error', fakeAsync(() => {
        component.appId = 42
        component.accessPointType = 'control'
        spyOn(servicesApi, 'getAccessPointKey').and.returnValue(throwError('error'))
        component.onAuthenticationKeyRequest()
        tick()
        expect(component.key).toBeUndefined()
        expect(component.isKeyFetched).toBeFalse()
    }))
})
