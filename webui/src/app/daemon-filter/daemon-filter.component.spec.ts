import { ComponentFixture, TestBed } from '@angular/core/testing'

import { DaemonFilterComponent } from './daemon-filter.component'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { ServicesService, SimpleDaemons } from '../backend'
import { of } from 'rxjs'

describe('DaemonFilterComponent', () => {
    let component: DaemonFilterComponent
    let fixture: ComponentFixture<DaemonFilterComponent>
    let servicesApi: ServicesService
    const differentDaemons: SimpleDaemons = {
        items: [
            {
                id: 1,
                name: 'dhcp4',
                machine: {
                    hostname: 'host_a',
                },
            },
            {
                id: 2,
                name: 'dhcp6',
                machine: {
                    hostname: 'host_b',
                },
            },
            {
                id: 3,
                name: 'named',
                machine: {
                    hostname: 'host_c',
                },
            },
        ],
        total: 3,
    }

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [DaemonFilterComponent],
            providers: [provideHttpClient(withInterceptorsFromDi())],
        }).compileComponents()

        fixture = TestBed.createComponent(DaemonFilterComponent)
        component = fixture.componentInstance
        servicesApi = fixture.debugElement.injector.get(ServicesService)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should have default label', () => {
        expect(component.label()).toEqual('Daemon (type or pick)')
    })

    it('should set label', () => {
        fixture.componentRef.setInput('label', 'test')
        fixture.detectChanges()
        expect(component.label()).toEqual('test')
    })

    it('should set matching daemon on init', () => {
        spyOn(servicesApi, 'getDaemonsDirectory').and.returnValue(of(differentDaemons as any))
        fixture.componentRef.setInput('daemonID', 2)
        component.ngOnInit()
        fixture.detectChanges()

        const expected = { ...differentDaemons.items[1], label: 'dhcp6@host_b' }
        expect(component.daemon).toEqual(expected)
    })

    it('should set matching daemon when input daemonID changes', () => {
        spyOn(servicesApi, 'getDaemonsDirectory').and.returnValue(of(differentDaemons as any))
        component.ngOnInit()
        fixture.detectChanges()

        expect(component.daemon).toBeFalsy()

        fixture.componentRef.setInput('daemonID', 3)
        fixture.detectChanges()

        const expected = { ...differentDaemons.items[2], label: 'named@host_c' }
        expect(component.daemon).toEqual(expected)
    })

    it('should set daemon to null when input daemonID changes and no match was found', () => {
        spyOn(servicesApi, 'getDaemonsDirectory').and.returnValue(of(differentDaemons as any))
        component.ngOnInit()
        fixture.detectChanges()

        expect(component.daemon).toBeFalsy()

        fixture.componentRef.setInput('daemonID', 9)
        fixture.detectChanges()

        expect(component.daemon).toBeFalsy()
    })

    it('should query all domains by default', () => {
        spyOn(servicesApi, 'getDaemonsDirectory').and.returnValue(of(differentDaemons as any))
        component.ngOnInit()
        fixture.detectChanges()
        expect(servicesApi.getDaemonsDirectory).toHaveBeenCalledOnceWith(undefined, undefined)
    })

    it('should query only dhcp domain', () => {
        spyOn(servicesApi, 'getDaemonsDirectory').and.returnValue(of(differentDaemons as any))
        fixture.componentRef.setInput('domain', 'dhcp')
        component.ngOnInit()
        fixture.detectChanges()
        expect(servicesApi.getDaemonsDirectory).toHaveBeenCalledOnceWith(undefined, 'dhcp')
    })

    it('should query only dns domain', () => {
        spyOn(servicesApi, 'getDaemonsDirectory').and.returnValue(of(differentDaemons as any))
        fixture.componentRef.setInput('domain', 'dns')
        component.ngOnInit()
        fixture.detectChanges()
        expect(servicesApi.getDaemonsDirectory).toHaveBeenCalledOnceWith(undefined, 'dns')
    })

    it('should set daemonID onValueChange', () => {
        spyOn(servicesApi, 'getDaemonsDirectory').and.returnValue(of(differentDaemons as any))
        component.ngOnInit()
        fixture.detectChanges()

        expect(component.daemon).toBeFalsy()
        expect(component.daemonID()).toBeFalsy()
        const newValue = { ...differentDaemons.items[2], label: 'named@host_c' }
        component.onValueChange(newValue)

        fixture.detectChanges()

        expect(component.daemonID()).toEqual(3)
    })

    it('should construct daemon label', () => {
        const resp: SimpleDaemons = {
            items: [
                {
                    id: 1,
                    name: 'dhcp4',
                    machine: {
                        hostname: 'host_a',
                    },
                },
                {
                    id: 2,
                    name: 'dhcp6',
                    machine: {
                        address: 'host_b_address',
                    },
                },
                {
                    id: 3,
                    name: 'named',
                    machineId: 8,
                },
            ],
            total: 3,
        }
        spyOn(servicesApi, 'getDaemonsDirectory').and.returnValue(of(resp as any))

        fixture.componentRef.setInput('daemonID', 1)
        component.ngOnInit()
        fixture.detectChanges()
        expect(component.daemon.label).toEqual('dhcp4@host_a')

        fixture.componentRef.setInput('daemonID', 2)
        component.ngOnInit()
        fixture.detectChanges()
        expect(component.daemon.label).toEqual('dhcp6@host_b_address')

        fixture.componentRef.setInput('daemonID', 3)
        component.ngOnInit()
        fixture.detectChanges()
        expect(component.daemon.label).toEqual('named@machine ID 8')
    })
})
