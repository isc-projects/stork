import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing'

import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { SubnetTabComponent } from './subnet-tab.component'
import { By } from '@angular/platform-browser'
import { ConfirmationService, MessageService } from 'primeng/api'
import { DHCPService } from '../backend'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { of, throwError } from 'rxjs'
import { provideRouter } from '@angular/router'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { AuthService } from '../auth.service'

describe('SubnetTabComponent', () => {
    let component: SubnetTabComponent
    let fixture: ComponentFixture<SubnetTabComponent>
    let dhcpApi: DHCPService
    let msgService: MessageService
    let confirmService: ConfirmationService
    let authService: AuthService

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [
                ConfirmationService,
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideNoopAnimations(),
                provideRouter([{ path: 'dhcp/subnets/:id', component: SubnetTabComponent }]),
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(SubnetTabComponent)
        component = fixture.componentInstance
        dhcpApi = fixture.debugElement.injector.get(DHCPService)
        confirmService = fixture.debugElement.injector.get(ConfirmationService)
        msgService = fixture.debugElement.injector.get(MessageService)
        authService = fixture.debugElement.injector.get(AuthService)
        spyOn(authService, 'superAdmin').and.returnValue(true)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display an IPv4 subnet', () => {
        component.subnet = {
            id: 1,
            subnet: '192.0.2.0/24',
            sharedNetwork: 'Fiber',
            addrUtilization: 30,
            stats: {
                'total-addresses': 240,
                'assigned-addresses': 70,
                'declined-addresses': 10,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    daemonId: 42,
                    daemonName: 'dhcp4',
                    pools: [
                        {
                            pool: '192.0.2.1-192.0.2.100',
                        },
                    ],
                    stats: {
                        'total-addresses': 240,
                        'assigned-addresses': 70,
                        'declined-addresses': 10,
                    },
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            cacheThreshold: 0.25,
                            cacheMaxAge: 1000,
                            options: [
                                {
                                    code: 1033,
                                },
                            ],
                            optionsHash: 'abc',
                        },
                        sharedNetworkLevelParameters: {
                            cacheThreshold: 0.3,
                            cacheMaxAge: 900,
                            options: [
                                {
                                    code: 1034,
                                },
                            ],
                            optionsHash: 'abc',
                        },
                        globalParameters: {
                            cacheThreshold: 0.29,
                            cacheMaxAge: 800,
                            options: [
                                {
                                    code: 1035,
                                },
                            ],
                            optionsHash: 'abc',
                        },
                    },
                    userContext: {
                        foo: 'user-context-is-here',
                    },
                },
            ],
        }
        component.ngOnInit()
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('Subnet 192.0.2.0/24 in shared network Fiber')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(6)

        expect(fieldsets[0].nativeElement.innerText).toContain('DHCP Servers Using the Subnet')
        expect(fieldsets[0].nativeElement.innerText).toContain('[42] DHCPv4')
        expect(fieldsets[0].nativeElement.innerText).toContain('12223')

        expect(fieldsets[1].nativeElement.innerText).toContain('Pools')
        expect(fieldsets[1].nativeElement.innerText).toContain('All Servers')

        const poolBar = fieldsets[1].query(By.css('app-address-pool-bar'))
        expect(poolBar).toBeTruthy()
        expect(poolBar.nativeElement.innerText).toContain('192.0.2.1-192.0.2.100')

        const charts = fieldsets[2].queryAll(By.css('p-chart'))
        expect(charts.length).toBe(1)

        // User contexts.
        expect(fieldsets[3].nativeElement.innerText).toContain('User Context')
        expect(fieldsets[3].nativeElement.innerText).toContain('foo')
        expect(fieldsets[3].nativeElement.innerText).toContain('user-context-is-here')

        expect(fieldsets[4].nativeElement.innerText).toContain('Cache Threshold')
        expect(fieldsets[4].nativeElement.innerText).toContain('0.25')
        expect(fieldsets[4].nativeElement.innerText).toContain('1000')

        // Ensure that the DHCP options are excluded from this list.
        expect(fieldsets[4].nativeElement.innerText).not.toContain('Options')
        expect(fieldsets[4].nativeElement.innerText).not.toContain('Options Hash')

        // DHCP options sit in their own fieldset.
        expect(fieldsets[5].nativeElement.innerText).toContain('DHCP Options')
        expect(fieldsets[5].nativeElement.innerText).toContain('1033')
    })

    it('should display an IPv4 subnet without pools', () => {
        component.subnet = {
            id: 1,
            subnet: '192.0.2.0/24',
            sharedNetwork: 'Fiber',
            addrUtilization: 30,
            stats: {
                'total-addresses': 240,
                'assigned-addresses': 70,
                'declined-addresses': 10,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    daemonId: 42,
                    daemonName: 'dhcp4',
                    stats: {
                        'total-addresses': 240,
                        'assigned-addresses': 70,
                        'declined-addresses': 10,
                    },
                },
            ],
        }
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('Subnet 192.0.2.0/24 in shared network Fiber')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(6)

        expect(fieldsets[0].nativeElement.innerText).toContain('DHCP Servers Using the Subnet')
        expect(fieldsets[0].nativeElement.innerText).toContain('[42] DHCPv4')
        expect(fieldsets[0].nativeElement.innerText).toContain('12223')

        expect(fieldsets[1].nativeElement.innerText).toContain('Pools')
        expect(fieldsets[1].nativeElement.innerText).toContain('All Servers')
        expect(fieldsets[1].nativeElement.innerText).toContain('No pools configured.')

        expect(fieldsets[3].nativeElement.innerText).toContain('No user context configured.')

        expect(fieldsets[4].nativeElement.innerText).toContain('No parameters configured.')

        expect(fieldsets[5].nativeElement.innerText).toContain('No options configured.')
    })

    it('should display an IPv6 subnet', () => {
        component.subnet = {
            id: 1,
            subnet: '2001:db8:1::/64',
            addrUtilization: 60,
            stats: {
                'total-nas': 1000,
                'assigned-nas': 30,
                'declined-nas': 10,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    daemonId: 42,
                    daemonName: 'dhcp6',
                    pools: [
                        {
                            pool: '2001:db8:1::2-2001:db8:1::786',
                        },
                    ],
                    stats: {
                        'total-nas': 1000,
                        'assigned-nas': 30,
                        'declined-nas': 10,
                    },
                },
            ],
        }
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('Subnet 2001:db8:1::/64')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(6)

        expect(fieldsets[0].nativeElement.innerText).toContain('DHCP Servers Using the Subnet')
        expect(fieldsets[0].nativeElement.innerText).toContain('[42] DHCPv6')
        expect(fieldsets[0].nativeElement.innerText).toContain('12223')

        expect(fieldsets[1].nativeElement.innerText).toContain('Pools')
        expect(fieldsets[1].nativeElement.innerText).toContain('All Servers')

        const poolBar = fieldsets[1].query(By.css('app-address-pool-bar'))
        expect(poolBar).toBeTruthy()
        expect(poolBar.nativeElement.innerText).toContain('2001:db8:1::2-2001:db8:1::786')

        const charts = fieldsets[2].queryAll(By.css('p-chart'))
        expect(charts.length).toBe(1)

        expect(fieldsets[3].nativeElement.innerText).toContain('No user context configured.')

        expect(fieldsets[4].nativeElement.innerText).toContain('No parameters configured.')

        expect(fieldsets[5].nativeElement.innerText).toContain('No options configured.')
    })

    it('should display an IPv6 subnet with address pools and prefixes', () => {
        component.subnet = {
            id: 1,
            subnet: '2001:db8:1::/64',
            addrUtilization: 88,
            pdUtilization: 60,
            stats: {
                'total-nas': 1024,
                'assigned-nas': 980,
                'declined-nas': 10,
                'total-pds': 500,
                'assigned-pds': 358,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    daemonId: 42,
                    daemonName: 'dhcp6',
                    pools: [
                        {
                            pool: '2001:db8:1::2-2001:db8:1::768',
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 80,
                        },
                    ],
                    stats: {
                        'total-nas': 1024,
                        'assigned-nas': 980,
                        'declined-nas': 10,
                        'total-pds': 500,
                        'assigned-pds': 358,
                    },
                },
            ],
        }
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('Subnet 2001:db8:1::/64')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(6)

        expect(fieldsets[0].nativeElement.innerText).toContain('DHCP Servers Using the Subnet')
        expect(fieldsets[0].nativeElement.innerText).toContain('[42] DHCPv6')
        expect(fieldsets[0].nativeElement.innerText).toContain('12223')

        expect(fieldsets[1].nativeElement.innerText).toContain('Pools')
        expect(fieldsets[1].nativeElement.innerText).toContain('All Servers')

        const poolBar = fieldsets[1].query(By.css('app-address-pool-bar'))
        expect(poolBar).toBeTruthy()
        expect(poolBar.nativeElement.innerText).toContain('2001:db8:1::2-2001:db8:1::768')

        const prefixBar = fieldsets[1].query(By.css('app-delegated-prefix-bar'))
        expect(prefixBar).toBeTruthy()
        expect(prefixBar.nativeElement.innerText).toContain('3000::')

        const charts = fieldsets[2].queryAll(By.css('p-chart'))
        expect(charts.length).toBe(2)

        expect(fieldsets[3].nativeElement.innerText).toContain('No user context configured.')

        expect(fieldsets[4].nativeElement.innerText).toContain('No parameters configured.')

        expect(fieldsets[5].nativeElement.innerText).toContain('No options configured.')
    })

    it('should display an IPv6 subnet with different fieldsets for different servers', () => {
        component.subnet = {
            id: 2,
            subnet: '2001:db8:1::/64',
            addrUtilization: 88,
            pdUtilization: 60,
            stats: {
                'total-nas': 1024,
                'assigned-nas': 980,
                'declined-nas': 10,
                'total-pds': 500,
                'assigned-pds': 358,
            },
            statsCollectedAt: '2023-06-05',
            localSubnets: [
                {
                    id: 12223,
                    daemonId: 42,
                    daemonName: 'dhcp6',
                    pools: [
                        {
                            pool: '2001:db8:1::2-2001:db8:1::768',
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::',
                            delegatedLength: 80,
                        },
                    ],
                    stats: {
                        'total-nas': 1024,
                        'assigned-nas': 500,
                        'declined-nas': 5,
                        'total-pds': 500,
                        'assigned-pds': 200,
                    },
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            cacheThreshold: 0.25,
                            options: [
                                {
                                    code: 3,
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.1'],
                                        },
                                    ],
                                },
                            ],
                            optionsHash: '123',
                        },
                        sharedNetworkLevelParameters: {
                            cacheThreshold: 0.3,
                        },
                        globalParameters: {
                            cacheThreshold: 0.29,
                        },
                    },
                    userContext: { foo: 'user-context-is-here' },
                },
                {
                    id: 25432,
                    daemonId: 43,
                    daemonName: 'dhcp6',
                    pools: [
                        {
                            pool: '2001:db8:1::2-2001:db8:1::768',
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/64',
                            delegatedLength: 80,
                        },
                        {
                            prefix: '3000:1::/64',
                            delegatedLength: 96,
                        },
                    ],
                    stats: {
                        'total-nas': 1024,
                        'assigned-nas': 480,
                        'declined-nas': 5,
                        'total-pds': 500,
                        'assigned-pds': 158,
                    },
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            cacheThreshold: 0.25,
                            options: [
                                {
                                    code: 3,
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.2'],
                                        },
                                    ],
                                },
                            ],
                            optionsHash: '234',
                        },
                        sharedNetworkLevelParameters: {
                            cacheThreshold: 0.3,
                        },
                        globalParameters: {
                            cacheThreshold: 0.29,
                        },
                    },
                },
            ],
        }
        fixture.detectChanges()

        expect(fixture.nativeElement.innerText).toContain('Subnet 2001:db8:1::/64')

        const fieldsets = fixture.debugElement.queryAll(By.css('p-fieldset'))
        expect(fieldsets.length).toBe(9)

        expect(fieldsets[0].nativeElement.innerText).toContain('DHCP Servers Using the Subnet')
        expect(fieldsets[0].nativeElement.innerText).toContain('[42] DHCPv6')
        expect(fieldsets[0].nativeElement.innerText).toContain('[43] DHCPv6')
        expect(fieldsets[0].nativeElement.innerText).toContain('12223')
        expect(fieldsets[0].nativeElement.innerText).toContain('25432')

        expect(fieldsets[1].nativeElement.innerText).toContain('Pools')
        expect(fieldsets[1].nativeElement.innerText).toContain('dhcp6')

        let poolBar = fieldsets[1].query(By.css('app-address-pool-bar'))
        expect(poolBar).toBeTruthy()
        expect(poolBar.nativeElement.innerText).toContain('2001:db8:1::2-2001:db8:1::768')

        let prefixBars = fieldsets[1].queryAll(By.css('app-delegated-prefix-bar'))
        expect(prefixBars.length).toBe(1)

        expect(fieldsets[2].nativeElement.innerText).toContain('Pools')
        expect(fieldsets[2].nativeElement.innerText).toContain('dhcp6')

        poolBar = fieldsets[2].query(By.css('app-address-pool-bar'))
        expect(poolBar).toBeTruthy()
        expect(poolBar.nativeElement.innerText).toContain('2001:db8:1::2-2001:db8:1::768')

        prefixBars = fieldsets[2].queryAll(By.css('app-delegated-prefix-bar'))
        expect(prefixBars.length).toBe(2)
        expect(prefixBars[0].nativeElement.innerText).toContain('3000::')
        expect(prefixBars[1].nativeElement.innerText).toContain('3000:1::')

        const charts = fieldsets[3].queryAll(By.css('p-chart'))
        expect(charts.length).toBe(6)

        expect(fieldsets[4].nativeElement.innerText).toContain('User Context')
        expect(fieldsets[4].nativeElement.innerText).toContain('foo')
        expect(fieldsets[4].nativeElement.innerText).toContain('user-context-is-here')
        expect(fieldsets[5].nativeElement.innerText).toContain('No user context configured.')

        expect(fieldsets[6].nativeElement.innerText).toContain('No parameters configured.')

        expect(fieldsets[7].nativeElement.innerText).toContain('DHCP Options')
        expect(fieldsets[7].nativeElement.innerText).toContain('dhcp6')
        expect(fieldsets[8].nativeElement.innerText).toContain('DHCP Options')
        expect(fieldsets[8].nativeElement.innerText).toContain('dhcp6')
    })

    it('should return shared network attributes for IPv6 subnet', () => {
        component.subnet = {
            subnet: '2001:db8:1::/64',
            sharedNetworkId: 123,
            sharedNetwork: 'foo',
        }
        expect(component.getSharedNetworkAttrs()).toEqual({
            id: 123,
            name: 'foo',
        })
    })

    it('should return shared network attributes for IPv4 subnet', () => {
        component.subnet = {
            subnet: '192.0.2.0/24',
            sharedNetworkId: 234,
            sharedNetwork: 'bar',
        }
        expect(component.getSharedNetworkAttrs()).toEqual({
            id: 234,
            name: 'bar',
        })
    })

    it('should display subnet delete button', () => {
        component.subnet = {
            id: 123,
            subnet: '2001:db8:1::/64',
            localSubnets: [
                {
                    id: 12223,
                    daemonName: 'dhcp6',
                },
            ],
        }
        fixture.detectChanges()

        fixture.detectChanges()
        const deleteBtn = fixture.debugElement.query(By.css('[label=Delete]'))
        expect(deleteBtn).toBeTruthy()

        // Simulate clicking on the button and make sure that the confirm dialog
        // has been displayed.
        spyOn(confirmService, 'confirm')
        deleteBtn.nativeElement.click()
        expect(confirmService.confirm).toHaveBeenCalled()
    })

    it('should emit an event indicating successful subnet deletion', fakeAsync(() => {
        const successResp: any = {}
        spyOn(dhcpApi, 'deleteSubnet').and.returnValue(of(successResp))
        spyOn(msgService, 'add')
        spyOn(component.subnetDelete, 'emit')

        // Delete the subnet.
        component.subnet = {
            id: 1,
        }
        component.deleteSubnet()
        tick()
        // Success message should be displayed.
        expect(msgService.add).toHaveBeenCalled()
        // An event should be called.
        expect(component.subnetDelete.emit).toHaveBeenCalledWith(component.subnet)
        // This flag should be cleared.
        expect(component.subnetDeleting).toBeFalse()
    }))

    it('should not emit an event when subnet deletion fails', fakeAsync(() => {
        spyOn(dhcpApi, 'deleteSubnet').and.returnValue(throwError({ status: 404 }))
        spyOn(msgService, 'add')
        spyOn(component.subnetDelete, 'emit')

        // Delete the host and receive an error.
        component.subnet = {
            id: 1,
        }
        component.deleteSubnet()
        tick()
        // Error message should be displayed.
        expect(msgService.add).toHaveBeenCalled()
        // The event shouldn't be emitted on error.
        expect(component.subnetDelete.emit).not.toHaveBeenCalledWith(component.subnet)
        // This flag should be cleared.
        expect(component.subnetDeleting).toBeFalse()
    }))

    it('should detect if any subnet has a user context', () => {
        // No local subnets.
        component.subnet = {}
        expect(component.hasUserContext).toBeFalse()

        component.subnet = { localSubnets: [] }
        expect(component.hasUserContext).toBeFalse()

        // Local subnets without user context.
        component.subnet = { localSubnets: [{}] }
        expect(component.hasUserContext).toBeFalse()

        component.subnet = { localSubnets: [{ userContext: null }] }
        expect(component.hasUserContext).toBeFalse()

        // Local subnets with user context.
        component.subnet = { localSubnets: [{ userContext: {} }] }
        expect(component.hasUserContext).toBeTrue()

        // Multiple local subnets.
        component.subnet = { localSubnets: [{ userContext: {} }, { userContext: {} }] }
        expect(component.hasUserContext).toBeTrue()

        component.subnet = { localSubnets: [{ userContext: {} }, { userContext: null }] }
        expect(component.hasUserContext).toBeTrue()

        component.subnet = { localSubnets: [{ userContext: null }, { userContext: {} }] }
        expect(component.hasUserContext).toBeTrue()

        component.subnet = { localSubnets: [{ userContext: null }, { userContext: null }] }
        expect(component.hasUserContext).toBeFalse()

        component.subnet = { localSubnets: [{}, {}] }
        expect(component.hasUserContext).toBeFalse()
    })

    it('should compare the user contexts of all local subnets', () => {
        // No local subnets.
        component.subnet = {}
        expect(component.allDaemonsHaveEqualUserContext()).toBeTrue()

        component.subnet = { localSubnets: [] }
        expect(component.allDaemonsHaveEqualUserContext()).toBeTrue()

        // Local subnets without user context.
        component.subnet = { localSubnets: [{}, {}] }
        expect(component.allDaemonsHaveEqualUserContext()).toBeTrue()

        // Local subnets with user context.
        component.subnet = { localSubnets: [{ userContext: {} }, { userContext: {} }] }
        expect(component.allDaemonsHaveEqualUserContext()).toBeTrue()

        component.subnet = { localSubnets: [{ userContext: {} }, {}] }
        expect(component.allDaemonsHaveEqualUserContext()).toBeFalse()

        component.subnet = { localSubnets: [{ userContext: { foo: 42 } }, { userContext: { foo: 42 } }] }
        expect(component.allDaemonsHaveEqualUserContext()).toBeTrue()

        component.subnet = {
            localSubnets: [{ userContext: { foo: 42 } }, { userContext: { foo: 42 } }, { userContext: { foo: 42 } }],
        }
        expect(component.allDaemonsHaveEqualUserContext()).toBeTrue()

        component.subnet = { localSubnets: [{ userContext: { foo: 42 } }, { userContext: { foo: 43 } }] }
        expect(component.allDaemonsHaveEqualUserContext()).toBeFalse()

        // Nested.
        component.subnet = {
            localSubnets: [{ userContext: { foo: { bar: 42 } } }, { userContext: { foo: { bar: 42 } } }],
        }
        expect(component.allDaemonsHaveEqualUserContext()).toBeTrue()

        component.subnet = {
            localSubnets: [{ userContext: { foo: { bar: 42 } } }, { userContext: { foo: { bar: 43 } } }],
        }
        expect(component.allDaemonsHaveEqualUserContext()).toBeFalse()

        // Array.
        component.subnet = { localSubnets: [{ userContext: { foo: [42] } }, { userContext: { foo: [42] } }] }

        expect(component.allDaemonsHaveEqualUserContext()).toBeTrue()

        component.subnet = { localSubnets: [{ userContext: { foo: [42] } }, { userContext: { foo: [43] } }] }
        expect(component.allDaemonsHaveEqualUserContext()).toBeFalse()
    })
})
