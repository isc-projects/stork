import { ComponentFixture, fakeAsync, TestBed, tick, waitForAsync } from '@angular/core/testing'

import { LeasesListTableComponent } from './leases-list-table.component'
import { Router, provideRouter } from '@angular/router'
import { ConfirmationService, MessageService, FilterMetadata } from 'primeng/api'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { InputNumber } from 'primeng/inputnumber'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { HttpErrorResponse, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { DHCPService, ServicesService } from '../backend'
import { By } from '@angular/platform-browser'
import { of, throwError } from 'rxjs'

describe('LeasesListTableComponent', () => {
    let component: LeasesListTableComponent
    let fixture: ComponentFixture<LeasesListTableComponent>
    let dhcpService: DHCPService
    let getLeaseListSpy: jasmine.Spy
    let router: Router

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                MessageService,
                ConfirmationService,
                provideNoopAnimations(),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideRouter([]),
                {
                    provide: ServicesService,
                    useValue: { getDaemonsDirectory: () => of({ items: [{ id: 1, label: 'daemon' }], total: 1 }) },
                },
            ],
        }).compileComponents()

        dhcpService = TestBed.inject(DHCPService)
        router = TestBed.inject(Router)
        getLeaseListSpy = spyOn(dhcpService, 'getLeaseList')
        spyOn(router, 'navigate')
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(LeasesListTableComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
        // Do not save table state between tests, because that makes tests unstable.
        spyOn(component.table, 'saveState').and.callFake(() => {})
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should not filter the table by numeric input with value zero', fakeAsync(() => {
        // Arrange
        const inputNumbers = fixture.debugElement.queryAll(By.directive(InputNumber))
        expect(inputNumbers).toBeTruthy()
        expect(inputNumbers.length).toEqual(3)

        // Act
        component.table.clear()
        tick()
        fixture.detectChanges()
        inputNumbers[0].componentInstance.handleOnInput(new InputEvent('input'), '', 0) // machineId
        tick(300)
        fixture.detectChanges()
        inputNumbers[1].componentInstance.handleOnInput(new InputEvent('input'), '', 0) // subnetId
        tick(300)
        fixture.detectChanges()
        inputNumbers[2].componentInstance.handleOnInput(new InputEvent('input'), '', 0) // keaSubnetId
        tick(300)
        fixture.detectChanges()

        // Assert
        expect(dhcpService.getLeaseList).toHaveBeenCalled()
        // Since zero is forbidden filter value for numeric inputs, we expect that minimum allowed value (i.e. 1) will be used.
        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: {
                machineId: 1,
                daemonId: null,
                subnetId: 1,
                localSubnetId: 1,
                text: null,
            },
        })
    }))

    it('should contain a refresh button', fakeAsync(() => {
        const refreshBtn = fixture.debugElement.query(By.css('[label="Refresh List"] button'))
        expect(refreshBtn).toBeTruthy()
        spyOn(component, 'loadData')

        getLeaseListSpy.and.returnValue(throwError(() => new HttpErrorResponse({ status: 404 })))
        refreshBtn.nativeElement.click()
        tick()
        fixture.detectChanges()
        expect(component.loadData).toHaveBeenCalled()
    }))

    it('should be filtered by subnetId', fakeAsync(() => {
      component.dataCollection = [
            {
                id: 1,
                daemonId: 8,
                daemonLabel: 'DHCPv6',
                cltt: 10,
                ipAddress: 'fe80::10',
                state: 1,
                subnetId: 24,
                validLifetime: 3600,
            },
        ]
        fixture.detectChanges()

        getLeaseListSpy.and.callThrough()

        component.filterTable(8, <FilterMetadata>component.table.filters['daemonId'])
        tick(300)
        fixture.detectChanges()

        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: {
                machineId: null,
                daemonId: 8,
                subnetId: null,
                localSubnetId: null,
                text: null,
            },
        })
    }))

    it('should be filtered by localSubnetId', fakeAsync(() => {
        component.dataCollection = [
            {
                id: 1,
                daemonId: 8,
                daemonLabel: 'DHCPv4',
                cltt: 11,
                ipAddress: '10.168.1.67',
                state: 1,
                subnetId: 27,
                validLifetime: 3600,
            },
        ]
        fixture.detectChanges()

        getLeaseListSpy.and.callThrough()

        component.filterTable(10, <FilterMetadata>component.table.filters['localSubnetId'])
        tick(300)
        fixture.detectChanges()

        expect(router.navigate).toHaveBeenCalledWith([], {
            queryParams: {
                machineId: null,
                daemonId: null,
                subnetId: null,
                localSubnetId: 10,
                text: null,
            },
        })
    }))
})
