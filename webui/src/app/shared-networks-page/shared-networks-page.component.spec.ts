import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, fakeAsync, tick, waitForAsync } from '@angular/core/testing'

import { SharedNetworksPageComponent } from './shared-networks-page.component'
import { provideRouter } from '@angular/router'
import { DHCPService, SharedNetwork } from '../backend'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { of, throwError } from 'rxjs'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { HttpErrorResponse, HttpEvent, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { ConfirmationService, MessageService } from 'primeng/api'

describe('SharedNetworksPageComponent', () => {
    let component: SharedNetworksPageComponent
    let fixture: ComponentFixture<SharedNetworksPageComponent>
    let dhcpService: DHCPService
    let messageService: MessageService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                ConfirmationService,
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideNoopAnimations(),
                provideRouter([
                    {
                        path: 'dhcp/shared-networks',
                        pathMatch: 'full',
                        redirectTo: 'dhcp/shared-networks/all',
                    },
                    {
                        path: 'dhcp/shared-networks/:id',
                        component: SharedNetworksPageComponent,
                    },
                ]),
            ],
        })

        dhcpService = TestBed.inject(DHCPService)
        messageService = TestBed.inject(MessageService)
        fixture = TestBed.createComponent(SharedNetworksPageComponent)
        component = fixture.componentInstance

        const fakeResponses: any[] = [
            {
                items: [
                    {
                        id: 1,
                        name: 'frog',
                        subnets: [
                            {
                                clientClass: 'class-00-00',
                                id: 5,
                                localSubnets: [
                                    {
                                        daemonId: 27,
                                        daemonName: 'dhcp4',
                                        id: 1,
                                        machineAddress: 'localhost',
                                        machineHostname: 'lv-pc',
                                        pools: [
                                            {
                                                pool: '1.0.0.4-1.0.255.254',
                                            },
                                        ],
                                    },
                                ],
                                subnet: '1.0.0.0/16',
                                statsCollectedAt: '2023-02-17T13:06:00.2134Z',
                                stats: {
                                    'assigned-addresses': '42',
                                    'total-addresses':
                                        '12345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                    'declined-addresses': '0',
                                },
                            },
                        ],
                        stats: {
                            'assigned-addresses':
                                '12345678901234567890123456789012345678901234567890123456789012345678901234567890',
                            'total-addresses':
                                '1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890',
                            'declined-addresses': '-2',
                        },
                        statsCollectedAt: '2022-01-19T12:10:22.513Z',
                    },
                ],
                total: 10496,
            },
            {
                items: [
                    {
                        id: 2,
                        name: 'frog-no-stats',
                        subnets: [
                            {
                                clientClass: 'class-00-00',
                                id: 5,
                                localSubnets: [
                                    {
                                        daemonId: 27,
                                        daemonName: 'dhcp4',
                                        id: 1,
                                        machineAddress: 'localhost',
                                        machineHostname: 'lv-pc',
                                        pools: [
                                            {
                                                pool: '1.0.0.4-1.0.255.254',
                                            },
                                        ],
                                    },
                                ],
                                subnet: '1.0.0.0/16',
                            },
                        ],
                        statsCollectedAt: '1970-01-01T12:00:00.0Z',
                    },
                ],
                total: 10496,
            },
            {
                items: [
                    {
                        id: 3,
                        name: 'cat',
                        subnets: [
                            // Subnet represented by the double utilization bar.
                            {
                                clientClass: 'class-00-00',
                                id: 5,
                                localSubnets: [
                                    {
                                        daemonId: 27,
                                        daemonName: 'dhcp4',
                                        id: 1,
                                        machineAddress: 'localhost',
                                        machineHostname: 'lv-pc',
                                    },
                                ],
                                subnet: 'fe80:1::/64',
                                statsCollectedAt: '2023-03-03T10:51:00.0000Z',
                                stats: {
                                    'assigned-nas': '42',
                                    'total-nas':
                                        '12345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                    'declined-nas': '0',
                                    'assigned-pds': '24',
                                    'total-pds':
                                        '9012345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                },
                                addrUtilization: 10,
                                pdUtilization: 15,
                            },
                            // Subnet represented by the single NA utilization bar.
                            {
                                clientClass: 'class-00-00',
                                id: 6,
                                localSubnets: [
                                    {
                                        daemonId: 27,
                                        daemonName: 'dhcp4',
                                        id: 1,
                                        machineAddress: 'localhost',
                                        machineHostname: 'lv-pc',
                                    },
                                ],
                                subnet: 'fe80:2::/64',
                                statsCollectedAt: '2023-03-03T10:51:00.0000Z',
                                stats: {
                                    'assigned-nas': '42',
                                    'total-nas':
                                        '12345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                    'declined-nas': '0',
                                    'assigned-pds': '0',
                                    'total-pds': '0',
                                },
                                addrUtilization: 20,
                                pdUtilization: 0,
                            },
                            // Subnet represented by the single PD utilization bar.
                            {
                                clientClass: 'class-00-00',
                                id: 7,
                                localSubnets: [
                                    {
                                        daemonId: 27,
                                        daemonName: 'dhcp4',
                                        id: 1,
                                        machineAddress: 'localhost',
                                        machineHostname: 'lv-pc',
                                    },
                                ],
                                subnet: 'fe80:3::/64',
                                statsCollectedAt: '2023-03-03T10:51:00.0000Z',
                                stats: {
                                    'assigned-nas': '0',
                                    'total-nas': '0',
                                    'declined-nas': '0',
                                    'assigned-pds': '0',
                                    'total-pds':
                                        '9012345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                },
                                addrUtilization: 0,
                                pdUtilization: 35,
                            },
                            // Subnet represented by the double utilization bar
                            {
                                clientClass: 'class-00-00',
                                id: 8,
                                localSubnets: [
                                    {
                                        daemonId: 27,
                                        daemonName: 'dhcp4',
                                        id: 2,
                                        machineAddress: 'localhost',
                                        machineHostname: 'lv-pc',
                                    },
                                ],
                                subnet: 'fe80:4::/64',
                                statsCollectedAt: '2023-03-03T10:51:00.0000Z',
                                stats: {
                                    'assigned-nas': '0',
                                    'total-nas': '0',
                                    'declined-nas': '0',
                                    'assigned-pds': '0',
                                    'total-pds': '0',
                                },
                                addrUtilization: 0,
                                pdUtilization: 0,
                            },
                        ],
                        statsCollectedAt: '1970-01-01T12:00:00.0Z',
                    },
                ],
                total: 10496,
            },
        ]
        const getNetworksSpy = spyOn(dhcpService, 'getSharedNetworks')
        // Prepare response when no filtering is applied.
        getNetworksSpy.withArgs(0, 10, null, null, null).and.returnValue(of(fakeResponses[0]))
        // Prepare response when shared networks are filtered by text to get an item without stats.
        getNetworksSpy.withArgs(0, 10, null, null, 'frog-no-stats').and.returnValue(of(fakeResponses[1]))
        // Prepare response when shared networks are filtered by text to get an item with 4 subnets.
        getNetworksSpy.withArgs(0, 10, null, null, 'cat').and.returnValue(of(fakeResponses[2]))

        spyOn(dhcpService, 'getSharedNetwork').and.returnValues(
            of(fakeResponses[0].items[0] as HttpEvent<SharedNetwork>)
        )

        fixture = TestBed.createComponent(SharedNetworksPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    }))

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('DHCP')
        expect(breadcrumbsComponent.items[1].label).toEqual('Shared Networks')
    })

    xit('should open and close tabs', fakeAsync(() => {
        // TODO: this test should be moved away from Karma tests.
        // component.openTabBySharedNetworkId(1)
        // tick()
        // fixture.detectChanges()
        //
        // expect(component.openedTabs.length).toBe(2)
        // expect(component.activeTabIndex).toBe(1)
        //
        // component.closeTabByIndex(1)
        //
        // expect(component.openedTabs.length).toBe(1)
        // expect(component.activeTabIndex).toBe(0)
        //
        // component.closeTabByIndex(0)
        //
        // expect(component.openedTabs.length).toBe(1)
        // expect(component.activeTabIndex).toBe(0)
    }))

    it('should call cancel transaction for new shared network', fakeAsync(() => {
        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpService, 'createSharedNetworkDelete').and.returnValue(of(okResp))

        component.callCreateNetworkDeleteTransaction(123)

        fixture.detectChanges()
        tick()

        expect(dhcpService.createSharedNetworkDelete).toHaveBeenCalledWith(123)
    }))

    it('should call cancel transaction for shared network update', fakeAsync(() => {
        const okResp: any = {
            status: 200,
        }

        spyOn(dhcpService, 'updateSharedNetworkDelete').and.returnValue(of(okResp))

        component.callUpdateNetworkDeleteTransaction(123, 321)

        tick()
        fixture.detectChanges()

        expect(dhcpService.updateSharedNetworkDelete).toHaveBeenCalledWith(123, 321)
    }))

    it('should display feedback when called cancel transaction for new shared network', fakeAsync(() => {
        spyOn(dhcpService, 'createSharedNetworkDelete').and.returnValue(
            throwError(() => new HttpErrorResponse({ status: 404, statusText: 'transaction not found' }))
        )

        const messageSpy = spyOn(messageService, 'add')

        component.callCreateNetworkDeleteTransaction(123)

        fixture.detectChanges()
        tick()

        expect(dhcpService.createSharedNetworkDelete).toHaveBeenCalledWith(123)
        expect(messageSpy).toHaveBeenCalledOnceWith(
            jasmine.objectContaining({
                summary: 'Failed to delete configuration transaction',
                severity: 'error',
                detail: 'Failed to delete configuration transaction: transaction not found',
            })
        )
    }))

    it('should display feedback when called cancel transaction for shared network update', fakeAsync(() => {
        spyOn(dhcpService, 'updateSharedNetworkDelete').and.returnValue(
            throwError(() => new HttpErrorResponse({ status: 404, statusText: 'transaction not found' }))
        )
        const messageSpy = spyOn(messageService, 'add')

        component.callUpdateNetworkDeleteTransaction(123, 321)

        tick()
        fixture.detectChanges()

        expect(dhcpService.updateSharedNetworkDelete).toHaveBeenCalledWith(123, 321)
        expect(messageSpy).toHaveBeenCalledOnceWith(
            jasmine.objectContaining({
                summary: 'Failed to delete configuration transaction',
                severity: 'error',
                detail: 'Failed to delete configuration transaction: transaction not found',
            })
        )
    }))

    xit('should close tab when shared network is deleted', fakeAsync(() => {
        // TODO: this test should be moved away from Karma tests.
        // tick()
        // fixture.detectChanges()
        //
        // // Open subnet tab.
        // component.openTabBySharedNetworkId(1)
        // fixture.detectChanges()
        // tick()
        // expect(component.openedTabs.length).toBe(2)
        //
        // // Simulate the notification that the shared network has been deleted.
        // component.onSharedNetworkDelete({
        //     id: 1,
        // })
        // fixture.detectChanges()
        // tick()
        //
        // // The main shared network tab should only be left.
        // expect(component.openedTabs.length).toBe(1)
    }))
})
