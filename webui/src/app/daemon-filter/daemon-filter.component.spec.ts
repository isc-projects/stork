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
                label: 'DHCPv4@host_a',
            },
            {
                id: 2,
                name: 'dhcp6',
                machine: {
                    hostname: 'host_b',
                },
                label: 'DHCPv6@host_b',
            },
            {
                id: 3,
                name: 'named',
                machine: {
                    hostname: 'host_c',
                },
                label: 'named@host_c',
            },
            {
                id: 93,
                name: 'netconf',
                machine: {
                    hostname: 'host_c',
                },
                label: 'NetConf@host_c',
            },
            {
                id: 94,
                name: 'd2',
                machine: {
                    hostname: 'host_c',
                },
                label: 'DDNS@host_c',
            },
            {
                id: 95,
                name: 'ca',
                machine: {
                    hostname: 'host_c',
                },
                label: 'CA@host_c',
            },
            {
                id: 96,
                name: 'pdns',
                machine: {
                    hostname: 'host_c',
                },
                label: 'pdns_server@host_c',
            },
        ],
        total: 7,
    }

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [DaemonFilterComponent],
            providers: [provideHttpClient(withInterceptorsFromDi())],
        }).compileComponents()

        fixture = TestBed.createComponent(DaemonFilterComponent)
        component = fixture.componentInstance
        servicesApi = fixture.debugElement.injector.get(ServicesService)
    })

    it('should create', () => {
        fixture.detectChanges()
        expect(component).toBeTruthy()
    })

    it('should have default label', () => {
        fixture.detectChanges()
        expect(component.label()).toEqual('Daemon (type or pick)')
    })

    it('should accept daemons depending on daemonNames input', () => {
        spyOn(servicesApi, 'getDaemonsDirectory').and.returnValue(of(differentDaemons as any))
        fixture.componentRef.setInput('daemonNames', ['dhcp4', 'dhcp6'])
        component.ngOnInit()
        fixture.detectChanges()

        expect(component.daemon).toBeFalsy()

        fixture.componentRef.setInput('daemonID', 1)
        fixture.detectChanges()
        expect(component.daemon.id).toEqual(1)

        fixture.componentRef.setInput('daemonID', 2)
        fixture.detectChanges()
        expect(component.daemon.id).toEqual(2)

        fixture.componentRef.setInput('daemonID', 3)
        fixture.detectChanges()
        expect(component.daemon).toBeFalsy()
    })

    it('should accept all daemons by default', () => {
        spyOn(servicesApi, 'getDaemonsDirectory').and.returnValue(of(differentDaemons as any))

        component.ngOnInit()
        fixture.detectChanges()

        expect(component.daemon).toBeFalsy()

        fixture.componentRef.setInput('daemonID', 1)
        fixture.detectChanges()
        expect(component.daemon.id).toEqual(1)

        fixture.componentRef.setInput('daemonID', 2)
        fixture.detectChanges()
        expect(component.daemon.id).toEqual(2)

        fixture.componentRef.setInput('daemonID', 3)
        fixture.detectChanges()
        expect(component.daemon.id).toEqual(3)

        fixture.componentRef.setInput('daemonID', 93)
        fixture.detectChanges()
        expect(component.daemon.id).toEqual(93)

        fixture.componentRef.setInput('daemonID', 94)
        fixture.detectChanges()
        expect(component.daemon.id).toEqual(94)

        fixture.componentRef.setInput('daemonID', 95)
        fixture.detectChanges()
        expect(component.daemon.id).toEqual(95)

        fixture.componentRef.setInput('daemonID', 96)
        fixture.detectChanges()
        expect(component.daemon.id).toEqual(96)
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

        const expected = { ...differentDaemons.items[1], listItemLabel: '[2] DHCPv6@host_b' }
        expect(component.daemon).toEqual(expected)
    })

    it('should set matching daemon when input daemonID changes', () => {
        spyOn(servicesApi, 'getDaemonsDirectory').and.returnValue(of(differentDaemons as any))
        component.ngOnInit()
        fixture.detectChanges()

        expect(component.daemon).toBeFalsy()

        fixture.componentRef.setInput('daemonID', 3)
        fixture.detectChanges()

        const expected = { ...differentDaemons.items[2], listItemLabel: '[3] named@host_c' }
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

    it('should set daemon to null when input daemonID changes and daemon name is not supported', () => {
        const resp: SimpleDaemons = {
            items: [...differentDaemons.items, { id: 4, name: 'ca', machineId: 7 }],
            total: 8,
        }
        spyOn(servicesApi, 'getDaemonsDirectory').and.returnValue(of(resp as any))
        fixture.componentRef.setInput('daemonNames', ['dhcp4'])
        component.ngOnInit()
        fixture.componentRef.setInput('daemonID', 4)
        fixture.detectChanges()

        expect(component.daemon).toBeFalsy()
    })

    it('should set daemonID onValueChange', () => {
        spyOn(servicesApi, 'getDaemonsDirectory').and.returnValue(of(differentDaemons as any))
        component.ngOnInit()
        fixture.detectChanges()

        expect(component.daemon).toBeFalsy()
        expect(component.daemonID()).toBeFalsy()
        const newValue = { ...differentDaemons.items[2], listItemLabel: 'named@host_c' }
        component.onValueChange(newValue)

        fixture.detectChanges()

        expect(component.daemonID()).toEqual(3)
    })

    it('should construct list item label', () => {
        const resp: SimpleDaemons = {
            items: [
                {
                    id: 1,
                    name: 'dhcp4',
                    machine: {
                        hostname: 'host_a',
                    },
                    label: 'DHCPv4@host_a',
                },
                {
                    id: 2,
                    name: 'dhcp6',
                    machine: {
                        address: 'host_b_address',
                    },
                    label: 'DHCPv6@host_b_address',
                },
                {
                    id: 3,
                    name: 'named',
                    machineId: 8,
                    label: 'named@machine ID 8',
                },
                {
                    id: 4,
                    name: 'pdns',
                    label: 'pdns_server@host_c',
                },
                {
                    id: 93,
                    name: 'netconf',
                    machine: {
                        hostname: 'host_c',
                    },
                    label: 'NetConf@host_c',
                },
                {
                    id: 94,
                    name: 'ca',
                    label: 'CA@host_c',
                },
                {
                    id: 95,
                    name: 'd2',
                    label: 'DDNS@host_c',
                },
            ],
            total: 7,
        }
        spyOn(servicesApi, 'getDaemonsDirectory').and.returnValue(of(resp as any))

        fixture.componentRef.setInput('daemonID', 1)
        component.ngOnInit()
        fixture.detectChanges()
        expect(component.daemon.listItemLabel).toEqual('[1] DHCPv4@host_a')

        fixture.componentRef.setInput('daemonID', 2)
        component.ngOnInit()
        fixture.detectChanges()
        expect(component.daemon.listItemLabel).toEqual('[2] DHCPv6@host_b_address')

        fixture.componentRef.setInput('daemonID', 3)
        component.ngOnInit()
        fixture.detectChanges()
        expect(component.daemon.listItemLabel).toEqual('[3] named@machine ID 8')

        fixture.componentRef.setInput('daemonID', 4)
        component.ngOnInit()
        fixture.detectChanges()
        expect(component.daemon.listItemLabel).toEqual('[4] pdns_server@host_c')

        fixture.componentRef.setInput('daemonID', 93)
        component.ngOnInit()
        fixture.detectChanges()
        expect(component.daemon.listItemLabel).toEqual('[93] NetConf@host_c')

        fixture.componentRef.setInput('daemonID', 94)
        component.ngOnInit()
        fixture.detectChanges()
        expect(component.daemon.listItemLabel).toEqual('[94] CA@host_c')

        fixture.componentRef.setInput('daemonID', 95)
        component.ngOnInit()
        fixture.detectChanges()
        expect(component.daemon.listItemLabel).toEqual('[95] DDNS@host_c')
    })
})
