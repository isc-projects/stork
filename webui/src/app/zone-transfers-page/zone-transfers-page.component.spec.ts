import { ComponentFixture, fakeAsync, TestBed, tick } from '@angular/core/testing'

import { ZoneTransfersPageComponent } from './zone-transfers-page.component'
import { ConfirmationService, FilterMetadata, MessageService } from 'primeng/api'
import { Daemon, DNSService, ServicesService } from '../backend'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { provideRouter, Router } from '@angular/router'
import { of, throwError } from 'rxjs'
import createSpyObj = jasmine.createSpyObj
import { TableState } from 'primeng/api'
import { TableLazyLoadEvent } from 'primeng/table'

describe('ZoneTransfersPage', () => {
    let component: ZoneTransfersPageComponent
    let fixture: ComponentFixture<ZoneTransfersPageComponent>
    let dnsService: jasmine.SpyObj<DNSService>
    let servicesService: jasmine.SpyObj<ServicesService>
    let messageService: jasmine.SpyObj<MessageService>
    let router: Router

    const machines = {
        items: [
            { id: 1, address: 'agent-bind9' },
            { id: 2, address: 'agent-bind9-2' },
        ],
        total: 2,
    } as any

    const zoneTransfers = {
        items: [
            {
                id: 1,
                createdAt: '2026-01-01T10:41:12Z',
                viewName: '_default',
                zoneName: 'zone.example.org',
                client: '192.0.2.1',
                server: '192.0.2.2',
                serial: 1234567890,
                status: 'completed',
                startedAt: '2026-01-01T10:40:27Z',
                completedAt: '2026-01-01T10:45:11Z',
                duration: '2h3m10.451s',
                bytesPerSecond: 12,
                message:
                    'AXFR completed: 79 messages, 24872 records, 1320233 bytes, 0.052 secs (25389096 bytes/sec) (serial 2026041600)',
                messagesCount: 79,
                recordsCount: 24872,
                bytesCount: 1320233,
            },
        ] as any,
        total: 1,
    } as any

    beforeEach(async () => {
        dnsService = createSpyObj('DNSService', ['getZoneTransferStates'])
        servicesService = createSpyObj('ServicesService', ['getMachinesDirectory'])
        messageService = createSpyObj('MessageService', ['add'])

        servicesService.getMachinesDirectory.and.returnValue(of(machines))

        dnsService.getZoneTransferStates.and.returnValue(of(zoneTransfers))

        await TestBed.configureTestingModule({
            providers: [
                { provide: MessageService, useValue: messageService },
                { provide: DNSService, useValue: dnsService },
                { provide: ServicesService, useValue: servicesService },
                ConfirmationService,
                provideNoopAnimations(),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideRouter([]),
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(ZoneTransfersPageComponent)
        component = fixture.componentInstance

        router = fixture.debugElement.injector.get(Router)
        spyOn(router, 'navigate')

        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should call API on init', fakeAsync(() => {
        expect(servicesService.getMachinesDirectory).toHaveBeenCalledWith([Daemon.NameEnum.Named])
        component.onLazyLoadZoneTransfers({})
        tick(300)
        expect(dnsService.getZoneTransferStates).toHaveBeenCalledWith(
            0,
            10,
            [],
            undefined,
            undefined,
            undefined,
            undefined,
            undefined,
            null,
            null
        )
        expect(component.machines).toEqual(machines.items)
        expect(component.zoneTransfersRows).toEqual(10)
        expect(component.zoneTransfersTotal).toEqual(1)
        expect(component.zoneTransfers?.length).toEqual(1)
        expect(component.zoneTransfers[0].zoneName).toEqual('zone.example.org')
    }))

    it('should report an error when API call for machines fails', fakeAsync(() => {
        servicesService.getMachinesDirectory.and.returnValue(throwError(() => new Error('Error')))
        component.ngOnInit()
        tick(300)
        expect(messageService.add).toHaveBeenCalledWith({
            severity: 'error',
            summary: 'Error retrieving machines',
            detail: 'Failed to retrieve machines information for filtering by primary and secondary: Error',
            life: 10000,
        })
    }))

    it('should report an error when API call for zone transfers fails', fakeAsync(() => {
        dnsService.getZoneTransferStates.and.returnValue(throwError(() => new Error('Error')))
        component.onLazyLoadZoneTransfers({} as TableLazyLoadEvent)
        tick(300)
        expect(messageService.add).toHaveBeenCalledWith({
            severity: 'error',
            summary: 'Error retrieving zone transfers',
            detail: 'Retrieving zone transfers information failed: Error',
            life: 10000,
        })
    }))

    it('should store rows per page for zones table', () => {
        component.storeZoneTransfersTableRowsPerPage(10)

        const stateString = localStorage.getItem('zone-transfers-table-state')
        expect(stateString).toBeTruthy()
        const state: TableState = JSON.parse(stateString ?? '{}')
        expect('rows' in state).toBeTrue()
        expect(state.rows).toEqual(10)
    })

    it('should clear filter', fakeAsync(() => {
        const filterConstraint = component.zoneTransfersTable.filters['zoneSerial'] as FilterMetadata
        component.filterZoneTransfersTable('1234567890', filterConstraint)
        tick(300)

        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: Object({
                zoneTransferStatus: null,
                zoneSerial: '1234567890',
                serverMachineId: null,
                clientMachineId: null,
                text: null,
                includeLocal: null,
            }),
        })
        component.clearFilter(filterConstraint)

        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: Object({
                zoneTransferStatus: null,
                zoneSerial: null,
                serverMachineId: null,
                clientMachineId: null,
                text: null,
                includeLocal: null,
            }),
        })
    }))
})
