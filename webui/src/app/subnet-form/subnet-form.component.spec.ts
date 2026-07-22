import { ComponentFixture, TestBed, fakeAsync, tick, flush } from '@angular/core/testing'

import { SubnetFormComponent } from './subnet-form.component'
import { FormArray, FormGroup, UntypedFormArray } from '@angular/forms'
import { HttpEvent, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { MessageService } from 'primeng/api'
import { Observable, of, throwError } from 'rxjs'
import { DHCPService, Subnet, UpdateSubnetBeginResponse } from '../backend'
import { AddressPoolForm, KeaSubnetParametersForm, PrefixPoolForm } from '../forms/subnet-set-form.service'
import { By } from '@angular/platform-browser'
import { provideRouter } from '@angular/router'

/**
 * Wraps response in HttpEvent type.
 */
function wrapInHttpResponse<T>(body: T): Observable<HttpEvent<T>> {
    return of(body as any)
}

describe('SubnetFormComponent', () => {
    let component: SubnetFormComponent
    let fixture: ComponentFixture<SubnetFormComponent>
    let dhcpApi: DHCPService
    let messageService: MessageService

    // TODO: This structure doesn't implement CreateSubnetBeginResponse.
    let cannedResponseBeginSubnet4: UpdateSubnetBeginResponse = {
        id: 123,
        subnet: {
            id: 123,
            subnet: '192.0.2.0/24',
            sharedNetwork: 'floor3',
            sharedNetworkId: 3,
            localSubnets: [
                {
                    id: 123,
                    daemonId: 1,
                    daemonLabel: 'DHCPv4@myhost.example.org',
                    pools: [
                        {
                            pool: '192.0.2.10-192.0.2.100',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                clientClasses: ['foo', 'bar'],
                                requireClientClasses: ['foo', 'bar'],
                                evaluateAdditionalClasses: ['foo', 'bar'],
                                options: [],
                                optionsHash: '',
                            },
                        },
                    ],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'random',
                            relay: {
                                ipAddresses: ['192.1.1.1'],
                            },
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 5,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.1'],
                                        },
                                    ],
                                    options: [],
                                    universe: 4,
                                },
                            ],
                            optionsHash: '123',
                        },
                    },
                    userContext: {
                        'subnet-name': 'server 1',
                    },
                },
                {
                    id: 123,
                    daemonId: 2,
                    daemonLabel: 'DHCPv4@remotehost.example.org',
                    pools: [
                        {
                            pool: '192.0.2.10-192.0.2.100',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: ['foo', 'bar'],
                                options: [],
                                optionsHash: '',
                            },
                        },
                    ],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'iterative',
                            relay: {
                                ipAddresses: ['192.1.1.1'],
                            },
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 5,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.2'],
                                        },
                                    ],
                                    options: [],
                                    universe: 4,
                                },
                            ],
                            optionsHash: '234',
                        },
                    },
                    userContext: {
                        'subnet-name': 'server 2',
                    },
                },
            ],
        },
        daemons: [
            {
                id: 1,
                name: 'dhcp4',
                label: 'DHCPv4@myhost.example.org',
                version: '2.7.4',
            },
            {
                id: 3,
                name: 'dhcp6',
                label: 'DHCPv6@myhost.example.org',
            },
            {
                id: 2,
                name: 'dhcp4',
                label: 'DHCPv4@yourhost.example.org',
                version: '2.7.3',
            },
            {
                id: 4,
                name: 'dhcp6',
                label: 'DHCPv6@yourhost.example.com',
            },
            {
                id: 5,
                name: 'dhcp6',
                label: 'DHCPv6@theirhost.example.com',
            },
        ],
        sharedNetworks4: [
            {
                id: 1,
                name: 'floor1',
                localSharedNetworks: [
                    {
                        daemonId: 1,
                        daemonLabel: 'DHCPv4@myhost.example.org',
                    },
                ],
            },
            {
                id: 2,
                name: 'floor2',
                localSharedNetworks: [
                    {
                        daemonId: 2,
                        daemonLabel: 'DHCPv4@yourhost.example.org',
                    },
                ],
            },
            {
                id: 3,
                name: 'floor3',
                localSharedNetworks: [
                    {
                        daemonId: 1,
                        daemonLabel: 'DHCPv4@myhost.example.org',
                    },
                    {
                        daemonId: 2,
                        daemonLabel: 'DHCPv4@yourhost.example.org',
                    },
                ],
            },
        ],
        sharedNetworks6: [],
    }

    // TODO: This structure doesn't implement CreateSubnetBeginResponse.
    let cannedResponseGroupedDaemons: UpdateSubnetBeginResponse = {
        id: 456,
        daemons: [
            {
                id: 10,
                name: 'dhcp4',
                label: 'zeta',
                serverTag: 'tag-a',
                backends: [
                    {
                        backendType: 'postgresql',
                        database: 'stork',
                        host: 'db.example.org',
                        port: 5432,
                        dataTypes: ['Config Backend'],
                    },
                ],
            },
            {
                id: 11,
                name: 'dhcp4',
                label: 'alpha',
                serverTag: 'tag-a',
                backends: [
                    {
                        backendType: 'postgresql',
                        database: 'stork',
                        host: 'db.example.org',
                        port: 5432,
                        dataTypes: ['Config Backend'],
                    },
                ],
            },
            {
                id: 12,
                name: 'dhcp4',
                label: 'beta',
                serverTag: 'tag-b',
                backends: [
                    {
                        backendType: 'postgresql',
                        database: 'stork',
                        host: 'db.example.org',
                        port: 5432,
                        dataTypes: ['Config Backend'],
                    },
                ],
            },
        ],
        sharedNetworks4: [],
        sharedNetworks6: [],
    }

    // TODO: This structure doesn't implement CreateSubnetBeginResponse.
    let cannedResponseSharedBackendUpdate: UpdateSubnetBeginResponse = {
        id: 501,
        subnet: {
            id: 901,
            subnet: '192.0.20.0/24',
            localSubnets: [
                {
                    id: 901,
                    daemonId: 1,
                    daemonLabel: 'dhcp4-a',
                    pools: [],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            options: [],
                        },
                    },
                },
            ],
        },
        daemons: [
            {
                id: 1,
                name: 'dhcp4',
                label: 'dhcp4-a',
                serverTag: 'tag-a',
                backends: [
                    {
                        backendType: 'postgresql',
                        database: 'stork',
                        host: 'db.example.org',
                        port: 5432,
                        dataTypes: ['Config Backend'],
                    },
                ],
            },
            {
                id: 4,
                name: 'dhcp4',
                label: 'dhcp4-b',
                serverTag: 'tag-b',
                backends: [
                    {
                        backendType: 'postgresql',
                        database: 'stork',
                        host: 'db.example.org',
                        port: 5432,
                        dataTypes: ['Config Backend'],
                    },
                ],
            },
        ],
        sharedNetworks4: [],
        sharedNetworks6: [],
    }

    // TODO: This structure doesn't implement CreateSubnetBeginResponse.
    let cannedResponseNoBackendUpdate: UpdateSubnetBeginResponse = {
        id: 502,
        subnet: {
            id: 777,
            subnet: '192.0.30.0/24',
            localSubnets: [
                {
                    id: 777,
                    daemonId: 1,
                    daemonLabel: 'dhcp4-1',
                    pools: [],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            options: [],
                        },
                    },
                },
                {
                    id: 777,
                    daemonId: 2,
                    daemonLabel: 'dhcp4-2',
                    pools: [],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            options: [],
                        },
                    },
                },
            ],
        },
        daemons: [
            {
                id: 1,
                name: 'dhcp4',
                label: 'dhcp4-1',
            },
            {
                id: 2,
                name: 'dhcp4',
                label: 'dhcp4-2',
            },
            {
                id: 3,
                name: 'dhcp4',
                label: 'dhcp4-3',
            },
        ],
        sharedNetworks4: [],
        sharedNetworks6: [],
    }

    // TODO: This structure doesn't implement CreateSubnetBeginResponse.
    let cannedResponseBeginSubnet6: UpdateSubnetBeginResponse = {
        id: 345,
        subnet: {
            id: 234,
            subnet: '2001:db8:1::/64',
            localSubnets: [
                {
                    id: 234,
                    daemonId: 3,
                    daemonLabel: 'DHCPv6@myhost.example.org',
                    pools: [
                        {
                            pool: '2001:db8:1::10-2001:db8:1::100',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                clientClasses: ['foo', 'bar'],
                                requireClientClasses: ['foo', 'bar'],
                                evaluateAdditionalClasses: ['foo', 'bar'],
                                options: [],
                                optionsHash: '',
                            },
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/48',
                            delegatedLength: 64,
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: [],
                                options: [],
                            },
                        },
                    ],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'random',
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 23,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv6-address',
                                            values: ['2001:db8:2::6789'],
                                        },
                                    ],
                                    options: [],
                                    universe: 6,
                                },
                            ],
                            optionsHash: '123',
                        },
                    },
                    userContext: {
                        'subnet-name': 'server 1',
                    },
                },
                {
                    id: 345,
                    daemonId: 4,
                    daemonLabel: 'DHCPv6@yourhost.example.org',
                    pools: [
                        {
                            pool: '2001:db8:1::10-2001:db8:1::100',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: ['foo', 'bar'],
                                options: [],
                                optionsHash: '',
                            },
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/48',
                            delegatedLength: 64,
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: [],
                                options: [],
                            },
                        },
                    ],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            pdAllocator: 'iterative',
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 23,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv6-address',
                                            values: ['2001:db8:2::6789'],
                                        },
                                    ],
                                    options: [],
                                    universe: 6,
                                },
                            ],
                            optionsHash: '123',
                        },
                    },
                    userContext: {
                        'subnet-name': 'server 2',
                    },
                },
            ],
        },
        daemons: [
            {
                id: 1,
                name: 'dhcp4',
                label: 'DHCPv4@myhost.example.org',
            },
            {
                id: 3,
                name: 'dhcp6',
                version: '2.7.4',
                label: 'DHCPv6@myhost.example.org',
            },
            {
                id: 2,
                name: 'dhcp4',
                label: 'DHCPv4@yourhost.example.org',
            },
            {
                id: 4,
                name: 'dhcp6',
                version: '2.7.3',
                label: 'DHCPv6@ourhost.example.org',
            },
            {
                id: 5,
                name: 'dhcp6',
                version: '2.7.3',
                label: 'DHCPv6@theirhost.example.org',
            },
        ],
        sharedNetworks4: [],
        sharedNetworks6: [
            {
                id: 1,
                name: 'floor1',
                localSharedNetworks: [
                    {
                        daemonId: 3,
                    },
                ],
            },
            {
                id: 2,
                name: 'floor2',
                localSharedNetworks: [
                    {
                        daemonId: 4,
                    },
                ],
            },
            {
                id: 3,
                name: 'floor3',
                localSharedNetworks: [
                    {
                        daemonId: 5,
                    },
                ],
            },
        ],
    }

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                provideNoopAnimations(),
                provideRouter([]),
                provideRouter([]),
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(SubnetFormComponent)
        component = fixture.componentInstance
        component.subnetId = 123
        dhcpApi = fixture.debugElement.injector.get(DHCPService)
        messageService = fixture.debugElement.injector.get(MessageService)
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should open a form for creating an IPv4 subnet', fakeAsync(() => {
        spyOn(dhcpApi, 'createSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseBeginSubnet4))
        component.subnetId = undefined
        component.ngOnInit()
        tick()
        fixture.detectChanges()
        expect(component.state).toBeTruthy()
        expect(component.state.preserved).toBeFalse()
        expect(component.state.transactionID).toBe(123)
        expect(component.state.group).toBeTruthy()
        expect(component.state.allDaemons.length).toBe(5)
        expect(component.state.filteredDaemons.length).toBe(5)
        expect(component.state.dhcpv4).toBeFalse()
        expect(component.state.dhcpv6).toBeFalse()
        expect(component.state.wizard).toBeTrue()

        expect(fixture.debugElement.query(By.css('[label="Proceed"]'))).toBeTruthy()
        expect(fixture.debugElement.query(By.css('[label="Cancel"]'))).toBeTruthy()
        expect(component.state.group.get('subnet').disabled).toBeFalse()
        expect(component.state.group.get('subnet').invalid).toBeTrue()

        component.state.group.get('subnet').setValue('192.0.2.0/24')
        expect(component.state.group.get('subnet').invalid).toBeFalse()

        component.onSubnetProceed()
        fixture.detectChanges()

        expect(component.state.group.get('subnet').disabled).toBeTrue()
        expect(component.state.wizard).toBeFalse()

        const selectedDaemonGroups = [0, 1]
        component.state.group.get('selectedDaemonGroups').setValue(selectedDaemonGroups)
        selectedDaemonGroups.forEach((index) => {
            component.onDaemonGroupsChange({
                itemValue: index,
            })
        })
        tick()
        fixture.detectChanges()

        // The daemons selection should be enabled, so there should be no helptip.
        expect(fixture.debugElement.query(By.css('[title="Help for disabled servers selection"]'))).toBeFalsy()

        // Set shared network. It should result in disabling the daemons selection.
        component.state.group.get('sharedNetwork').setValue(3)
        component.onSharedNetworkChange({
            value: 3,
        })
        fixture.detectChanges()
        expect(component.state.group.get('selectedDaemonGroups')?.disabled).toBeTrue()

        // It should also display the helptip.
        expect(fixture.debugElement.query(By.css('[title="Help for disabled servers selection"]'))).toBeTruthy()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'createSubnetSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const subnet = {
            subnet: '192.0.2.0/24',
            sharedNetworkId: 3,
            sharedNetwork: 'floor3',
            localSubnets: [
                {
                    daemonId: 1,
                    pools: [],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            options: [],
                        },
                    },
                },
                {
                    daemonId: 2,
                    pools: [],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            options: [],
                        },
                    },
                },
            ],
        }

        expect(dhcpApi.createSubnetSubmit).toHaveBeenCalledWith(component.state.transactionID, subnet)
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should group daemons by server tag and sort labels', fakeAsync(() => {
        spyOn(dhcpApi, 'createSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseGroupedDaemons))
        component.subnetId = undefined
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.state.allDaemons.length).toBe(2)
        expect(component.state.allDaemons[0].label).toBe('tag-a (zeta) + 1 more')
        expect(component.state.allDaemons[0].daemons.map((daemon) => daemon.id)).toEqual([10, 11])
        expect(component.state.allDaemons[1].label).toBe('tag-b (beta)')
        expect(component.state.allDaemons[1].daemons.map((daemon) => daemon.id)).toEqual([12])
    }))

    it('should open a form for creating an IPv6 subnet', fakeAsync(() => {
        spyOn(dhcpApi, 'createSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseBeginSubnet4))
        component.subnetId = undefined
        component.ngOnInit()
        tick()
        fixture.detectChanges()
        expect(component.state).toBeTruthy()
        expect(component.state.preserved).toBeFalse()
        expect(component.state.transactionID).toBe(123)
        expect(component.state.group).toBeTruthy()
        expect(component.state.allDaemons.length).toBe(5)
        expect(component.state.filteredDaemons.length).toBe(5)
        expect(component.state.dhcpv4).toBeFalse()
        expect(component.state.dhcpv6).toBeFalse()
        expect(component.state.wizard).toBeTrue()

        const button = fixture.debugElement.query(By.css('[label="Proceed"]'))
        expect(button).toBeTruthy()
        expect(component.state.group.get('subnet').disabled).toBeFalse()
        expect(component.state.group.get('subnet').invalid).toBeTrue()

        component.state.group.get('subnet').setValue('2001:db8:3::/64')
        expect(component.state.group.get('subnet').invalid).toBeFalse()

        component.onSubnetProceed()
        fixture.detectChanges()

        expect(component.state.group.get('subnet').disabled).toBeTrue()
        expect(component.state.wizard).toBeFalse()

        const selectedDaemonGroups = [2, 3]
        component.state.group.get('selectedDaemonGroups').setValue(selectedDaemonGroups)
        selectedDaemonGroups.forEach((index) => {
            component.onDaemonGroupsChange({
                itemValue: index,
            })
        })
        tick()

        // Ensure there is no shared network selected.
        component.onSharedNetworkChange({
            value: null,
        })
        fixture.detectChanges()

        // Since shared network is not selected, the daemons selection should
        // be enabled.
        expect(component.state.group.get('selectedDaemonGroups')?.disabled).toBeFalse()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'createSubnetSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const subnet = {
            subnet: '2001:db8:3::/64',
            sharedNetworkId: null,
            localSubnets: [
                {
                    daemonId: 3,
                    pools: [],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            options: [],
                        },
                    },
                },
                {
                    daemonId: 4,
                    pools: [],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            options: [],
                        },
                    },
                },
            ],
        }

        expect(dhcpApi.createSubnetSubmit).toHaveBeenCalledWith(component.state.transactionID, subnet)
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should reuse local subnet ID for a daemon sharing config backend', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseSharedBackendUpdate))
        component.subnetId = 901
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        const daemon1GroupIndex = component.state.allDaemons.find((group) =>
            group.daemons.some((daemon) => daemon.id === 1)
        )?.index
        const daemon4GroupIndex = component.state.allDaemons.find((group) =>
            group.daemons.some((daemon) => daemon.id === 4)
        )?.index
        expect(daemon1GroupIndex).not.toBeUndefined()
        expect(daemon4GroupIndex).not.toBeUndefined()
        const selectedDaemonGroups = [daemon1GroupIndex, daemon4GroupIndex].filter(
            (groupIndex): groupIndex is number => groupIndex != null
        )

        component.state.group.get('selectedDaemonGroups')?.setValue(selectedDaemonGroups)
        component.onDaemonGroupsChange({
            itemValue: daemon4GroupIndex,
        })
        tick()
        fixture.detectChanges()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'updateSubnetSubmit').and.returnValue(of(okResp))
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const submittedSubnet = (dhcpApi.updateSubnetSubmit as jasmine.Spy).calls.mostRecent().args[2] as Subnet
        const daemon1Subnet = submittedSubnet.localSubnets.find((localSubnet) => localSubnet.daemonId === 1)
        const daemon4Subnet = submittedSubnet.localSubnets.find((localSubnet) => localSubnet.daemonId === 4)
        expect(daemon1Subnet?.id).toBe(901)
        expect(daemon4Subnet?.id).toBe(901)
    }))

    it('should set a non-zero local subnet ID for a newly selected daemon without backend data', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseNoBackendUpdate))
        component.subnetId = 777
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        const daemon1GroupIndex = component.state.allDaemons.find((group) =>
            group.daemons.some((daemon) => daemon.id === 1)
        )?.index
        const daemon2GroupIndex = component.state.allDaemons.find((group) =>
            group.daemons.some((daemon) => daemon.id === 2)
        )?.index
        const daemon3GroupIndex = component.state.allDaemons.find((group) =>
            group.daemons.some((daemon) => daemon.id === 3)
        )?.index
        expect(daemon1GroupIndex).not.toBeUndefined()
        expect(daemon2GroupIndex).not.toBeUndefined()
        expect(daemon3GroupIndex).not.toBeUndefined()
        const selectedDaemonGroups = [daemon1GroupIndex, daemon2GroupIndex, daemon3GroupIndex].filter(
            (groupIndex): groupIndex is number => groupIndex != null
        )

        component.state.group.get('selectedDaemonGroups')?.setValue(selectedDaemonGroups)
        component.onDaemonGroupsChange({
            itemValue: daemon3GroupIndex,
        })
        tick()
        fixture.detectChanges()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'updateSubnetSubmit').and.returnValue(of(okResp))
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const submittedSubnet = (dhcpApi.updateSubnetSubmit as jasmine.Spy).calls.mostRecent().args[2] as Subnet
        const daemon3Subnet = submittedSubnet.localSubnets.find((localSubnet) => localSubnet.daemonId === 3)
        expect(daemon3Subnet?.id).toBe(777)
    }))

    it('should open a form for updating IPv4 subnet', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseBeginSubnet4))
        component.subnetId = 123
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.state).toBeTruthy()
        expect(component.state.preserved).toBeFalse()
        expect(component.state.transactionID).toBe(123)
        expect(component.state.group).toBeTruthy()
        expect(component.state.allDaemons.length).toBe(5)
        expect(component.state.filteredDaemons.length).toBe(2)
        expect(component.state.dhcpv4).toBeTrue()
        expect(component.state.dhcpv6).toBeFalse()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'updateSubnetSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const subnet = {
            id: 123,
            subnet: '192.0.2.0/24',
            sharedNetworkId: 3,
            sharedNetwork: 'floor3',
            localSubnets: [
                {
                    id: 123,
                    daemonId: 1,
                    pools: [
                        {
                            pool: '192.0.2.10-192.0.2.100',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                clientClasses: ['foo', 'bar'],
                                requireClientClasses: ['foo', 'bar'],
                                evaluateAdditionalClasses: ['foo', 'bar'],
                                options: [],
                            },
                        },
                    ],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'random',
                            relay: {
                                ipAddresses: ['192.1.1.1'],
                            },
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 5,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.1'],
                                        },
                                    ],
                                    options: [],
                                    universe: 4,
                                    unknown: {},
                                },
                            ],
                        },
                    },
                    userContext: {
                        'subnet-name': 'server 1',
                    },
                },
                {
                    id: 123,
                    daemonId: 2,
                    pools: [
                        {
                            pool: '192.0.2.10-192.0.2.100',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: ['foo', 'bar'],
                                options: [],
                            },
                        },
                    ],
                    prefixDelegationPools: [],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'iterative',
                            relay: {
                                ipAddresses: ['192.1.1.1'],
                            },
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 5,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv4-address',
                                            values: ['192.0.2.2'],
                                        },
                                    ],
                                    options: [],
                                    universe: 4,
                                    unknown: {},
                                },
                            ],
                        },
                    },
                    userContext: {
                        'subnet-name': 'server 2',
                    },
                },
            ],
        }

        expect(dhcpApi.updateSubnetSubmit).toHaveBeenCalledWith(
            component.subnetId,
            component.state.transactionID,
            subnet
        )
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should open a form for updating IPv6 subnet', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.state).toBeTruthy()
        expect(component.state.preserved).toBeFalse()
        expect(component.state.transactionID).toBe(345)
        expect(component.state.group).toBeTruthy()
        expect(component.state.allDaemons.length).toBe(5)
        expect(component.state.filteredDaemons.length).toBe(3)
        expect(component.state.dhcpv4).toBeFalse()
        expect(component.state.dhcpv6).toBeTrue()

        const okResp: any = {
            status: 200,
        }
        spyOn(dhcpApi, 'updateSubnetSubmit').and.returnValue(of(okResp))
        spyOn(component.formSubmit, 'emit')
        spyOn(messageService, 'add')
        component.onSubmit()
        tick()
        fixture.detectChanges()

        const subnet: Subnet = {
            id: 234,
            subnet: '2001:db8:1::/64',
            sharedNetworkId: null,
            localSubnets: [
                {
                    id: 234,
                    daemonId: 3,
                    pools: [
                        {
                            pool: '2001:db8:1::10-2001:db8:1::100',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                clientClasses: ['foo', 'bar'],
                                requireClientClasses: ['foo', 'bar'],
                                evaluateAdditionalClasses: ['foo', 'bar'],
                                options: [],
                            },
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/48',
                            delegatedLength: 64,
                            excludedPrefix: null,
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                clientClasses: [],
                                requireClientClasses: [],
                                evaluateAdditionalClasses: [],
                                options: [],
                            },
                        },
                    ],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            allocator: 'random',
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 23,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv6-address',
                                            values: ['2001:db8:2::6789'],
                                        },
                                    ],
                                    options: [],
                                    universe: 6,
                                    unknown: {},
                                },
                            ],
                        },
                    },
                    userContext: {
                        'subnet-name': 'server 1',
                    },
                },
                {
                    id: 345,
                    daemonId: 4,
                    pools: [
                        {
                            pool: '2001:db8:1::10-2001:db8:1::100',
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: ['foo', 'bar'],
                                options: [],
                            },
                        },
                    ],
                    prefixDelegationPools: [
                        {
                            prefix: '3000::/48',
                            delegatedLength: 64,
                            excludedPrefix: null,
                            keaConfigPoolParameters: {
                                clientClass: 'foo',
                                requireClientClasses: [],
                                options: [],
                            },
                        },
                    ],
                    keaConfigSubnetParameters: {
                        subnetLevelParameters: {
                            pdAllocator: 'iterative',
                            options: [
                                {
                                    alwaysSend: true,
                                    code: 23,
                                    encapsulate: '',
                                    fields: [
                                        {
                                            fieldType: 'ipv6-address',
                                            values: ['2001:db8:2::6789'],
                                        },
                                    ],
                                    options: [],
                                    universe: 6,
                                    unknown: {},
                                },
                            ],
                        },
                    },
                    userContext: {
                        'subnet-name': 'server 2',
                    },
                },
            ],
        }

        expect(dhcpApi.updateSubnetSubmit).toHaveBeenCalledWith(
            component.subnetId,
            component.state.transactionID,
            subnet
        )
        expect(component.formSubmit.emit).toHaveBeenCalled()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should initialize the form controls for an IPv4 subnet', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseBeginSubnet4))
        component.subnetId = 123
        component.ngOnInit()
        tick()
        // We cannot use contains() function here because it returns false for
        // disabled controls.
        expect(component.state).toBeTruthy()
        expect(component.state.group).toBeTruthy()
        expect(component.state.group.get('subnet')).toBeTruthy()
        expect(component.state.group.get('sharedNetwork')).toBeTruthy()
        expect(component.state.group.get('pools')).toBeTruthy()
        expect(component.state.group.contains('parameters')).toBeTrue()
        expect(component.state.group.contains('options')).toBeTrue()

        expect(component.state.group.get('subnet').value).toBe('192.0.2.0/24')
        expect(component.state.group.get('sharedNetwork').value).toBe(3)

        const pools = component.state.group.get('pools') as FormArray<FormGroup<AddressPoolForm>>
        expect(pools).toBeTruthy()
        expect(pools.length).toBe(1)
        expect(pools.get('0.range.start')?.value).toBe('192.0.2.10')
        expect(pools.get('0.range.end')?.value).toBe('192.0.2.100')

        const parameters = component.state.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters).toBeTruthy()
        expect(parameters.get('allocator.unlocked')?.value).toBeTrue()
        expect(parameters.get('allocator.values')?.value).toEqual(['random', 'iterative'])

        const options = component.state.group.get('options')
        expect(options).toBeTruthy()
        expect(options.get('unlocked')?.value).toBeTrue()
        const data = options.get('data') as UntypedFormArray
        expect(data).toBeTruthy()
        expect(data.length).toBe(2)
        expect(data.get('0.0.optionCode')?.value).toBe(5)
        expect(data.get('1.0.optionCode')?.value).toBe(5)

        const userContextsNames = component.state.group.get('userContexts.names') as UntypedFormArray
        expect(userContextsNames).toBeTruthy()
        expect(userContextsNames.length).toBe(2)
        expect(userContextsNames.get('0').value).toBe('server 1')
        expect(userContextsNames.get('1').value).toBe('server 2')

        const userContexts = component.state.group.get('userContexts.contexts') as UntypedFormArray
        expect(userContexts).toBeTruthy()
        expect(userContexts.length).toBe(2)
        expect(userContexts.get('0').value).toEqual({
            'subnet-name': 'server 1',
        })
        expect(userContexts.get('1').value).toEqual({
            'subnet-name': 'server 2',
        })
    }))

    it('should initialize the form controls for an IPv6 subnet', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()
        // We cannot use contains() function here because it returns false for
        // disabled controls.
        expect(component.state).toBeTruthy()
        expect(component.state.group).toBeTruthy()
        expect(component.state.group.get('subnet')).toBeTruthy()
        expect(component.state.group.get('sharedNetwork')).toBeTruthy()
        expect(component.state.group.get('pools')).toBeTruthy()
        expect(component.state.group.contains('parameters')).toBeTrue()
        expect(component.state.group.contains('options')).toBeTrue()

        expect(component.state.group.get('subnet').value).toBe('2001:db8:1::/64')
        expect(component.state.group.get('sharedNetwork').value).toBeFalsy()

        const pools = component.state.group.get('pools') as FormArray<FormGroup<AddressPoolForm>>
        expect(pools).toBeTruthy()
        expect(pools.length).toBe(1)
        expect(pools.get('0.range.start')?.value).toBe('2001:db8:1::10')
        expect(pools.get('0.range.end')?.value).toBe('2001:db8:1::100')

        const prefixPools = component.state.group.get('prefixPools') as FormArray<FormGroup<PrefixPoolForm>>
        expect(prefixPools).toBeTruthy()
        expect(prefixPools.length).toBe(1)
        expect(prefixPools.get('0.prefixes.prefix')?.value).toBe('3000::/48')
        expect(prefixPools.get('0.prefixes.delegatedLength')?.value).toBe(64)

        const parameters = component.state.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters).toBeTruthy()
        expect(parameters.get('allocator.unlocked')?.value).toBeTrue()
        expect(parameters.get('allocator.values')?.value).toEqual(['random', null])

        const options = component.state.group.get('options')
        expect(options).toBeTruthy()
        expect(options.get('unlocked')?.value).toBeFalse()
        const data = options.get('data') as UntypedFormArray
        expect(data?.length).toBe(2)
        expect(data.get('0.0.optionCode')?.value).toBe(23)
        expect(data.get('1.0.optionCode')?.value).toBe(23)

        const userContextsNames = component.state.group.get('userContexts.names') as UntypedFormArray
        expect(userContextsNames).toBeTruthy()
        expect(userContextsNames.length).toBe(2)
        expect(userContextsNames.get('0').value).toBe('server 1')
        expect(userContextsNames.get('1').value).toBe('server 2')

        const userContexts = component.state.group.get('userContexts.contexts') as UntypedFormArray
        expect(userContexts).toBeTruthy()
        expect(userContexts.length).toBe(2)
        expect(userContexts.get('0').value).toEqual({
            'subnet-name': 'server 1',
        })
        expect(userContexts.get('1').value).toEqual({
            'subnet-name': 'server 2',
        })
    }))

    it('should return a valid pool header', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()
        expect(component.getPoolHeader(0)).toEqual(['2001:db8:1::10-2001:db8:1::100', true])
        expect(component.getPoolHeader(1)).toEqual(['New Pool', false])
    }))

    it('should return a valid prefix pool header', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()
        expect(component.getPrefixPoolHeader(0)).toEqual(['3000::/48', true])
        expect(component.getPrefixPoolHeader(1)).toEqual(['New Pool', false])
    }))

    it('should present the pool in accordion', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseBeginSubnet4))
        component.subnetId = 123
        component.ngOnInit()
        tick()
        fixture.detectChanges()
        tick()

        const poolsPanel = fixture.debugElement.query(By.css('[legend="Pools"]'))
        expect(poolsPanel).toBeTruthy()

        const poolPanel = poolsPanel.query(By.css('p-accordion'))
        expect(poolsPanel).toBeTruthy()
        expect(poolPanel.nativeElement.innerText).toContain('192.0.2.10-192.0.2.100')
    }))

    it('should present the prefix pool in accordion', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()
        fixture.detectChanges()
        tick()

        const poolsPanel = fixture.debugElement.query(By.css('[legend="Prefix Delegation Pools"]'))
        expect(poolsPanel).toBeTruthy()

        const poolPanel = poolsPanel.query(By.css('p-accordion'))
        expect(poolsPanel).toBeTruthy()
        expect(poolPanel.nativeElement.innerText).toContain('3000::/48')
    }))

    it('should return correct server tag severity', () => {
        expect(component.getServerTagSeverity(0)).toBe('success')
        expect(component.getServerTagSeverity(1)).toBe('warn')
        expect(component.getServerTagSeverity(2)).toBe('danger')
        expect(component.getServerTagSeverity(3)).toBe('info')
        expect(component.getServerTagSeverity(4)).toBe('info')
    })

    it('should remove the form for the unselected server', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseBeginSubnet4))
        component.subnetId = 123
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        // Expand the tab.
        const tab = fixture.debugElement.query(By.css('.p-accordionheader'))
        expect(tab).toBeTruthy()
        tab.nativeElement.click()
        fixture.detectChanges()

        expect(component.addressPoolComponents.length).toBe(1)
        spyOn(component.addressPoolComponents.get(0), 'handleDaemonGroupsChange').and.callThrough()

        component.state.group.get('selectedDaemonGroups').setValue([1])
        component.onDaemonGroupsChange({
            itemValue: 0,
        })
        tick()
        fixture.detectChanges()

        expect(component.addressPoolComponents.get(0).handleDaemonGroupsChange).toHaveBeenCalledOnceWith(0)
        expect(component.addressPoolComponents.get(0).selectableGroups.length).toBe(1)
        expect(component.addressPoolComponents.get(0).selectableGroups[0].index).toBe(1)

        const options = component.state.group.get('options.data') as UntypedFormArray
        expect(options).toBeTruthy()
        expect(options.length).toBe(1)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('192.0.2.2')

        const parameters = component.state.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters.get('allocator.unlocked')?.value).toBeFalse()
        expect((parameters.get('allocator.values') as UntypedFormArray).length).toBe(1)
        expect(parameters.get('allocator.values.0')?.value).toBe('iterative')

        const userContextsNames = component.state.group.get('userContexts.names') as UntypedFormArray
        expect(userContextsNames).toBeTruthy()
        expect(userContextsNames.length).toBe(1)
        expect(userContextsNames.get('0').value).toBe('server 2')

        const userContexts = component.state.group.get('userContexts.contexts') as UntypedFormArray
        expect(userContexts).toBeTruthy()
        expect(userContexts.length).toBe(1)
        expect(userContexts.get('0').value).toEqual({
            'subnet-name': 'server 2',
        })

        expect(component.state.servers.length).toBe(1)
        expect(component.state.servers[0]).toBe('DHCPv4@yourhost.example.org')

        flush()
        // TODO: this test should be probably moved away from Karma tests. flush() is saving us from: Error: 11 timer(s) still in the queue.
    }))

    it('should create the form for the selected server', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        // Expand the tab.
        const tab = fixture.debugElement.query(By.css('.p-accordionheader'))
        expect(tab).toBeTruthy()
        tab.nativeElement.click()
        tick()
        fixture.detectChanges()

        expect(component.addressPoolComponents.length).toBe(1)
        spyOn(component.addressPoolComponents.get(0), 'handleDaemonGroupsChange').and.callThrough()

        component.state.group.get('selectedDaemonGroups').setValue([2, 3, 4])
        component.onDaemonGroupsChange({
            itemValue: 4,
        })
        tick()
        fixture.detectChanges()

        expect(component.addressPoolComponents.get(0).handleDaemonGroupsChange).toHaveBeenCalledOnceWith(4)
        expect(component.addressPoolComponents.get(0).selectableGroups.length).toBe(3)

        const options = component.state.group.get('options.data') as UntypedFormArray
        expect(options.length).toBe(3)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('2001:db8:2::6789')
        expect(options.get('1.0.optionFields.0.control')?.value).toBe('2001:db8:2::6789')
        expect(options.get('2.0.optionFields.0.control')?.value).toBe('2001:db8:2::6789')

        const parameters = component.state.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters.get('pdAllocator.unlocked')?.value).toBeTrue()
        expect((parameters.get('pdAllocator.values') as UntypedFormArray).length).toBe(3)
        expect(parameters.get('pdAllocator.values.0')?.value).toBeFalsy()
        expect(parameters.get('pdAllocator.values.1')?.value).toBe('iterative')
        expect(parameters.get('pdAllocator.values.2')?.value).toBeFalsy()

        const userContextsNames = component.state.group.get('userContexts.names') as UntypedFormArray
        expect(userContextsNames).toBeTruthy()
        expect(userContextsNames.length).toBe(3)
        expect(userContextsNames.get('0').value).toBe('server 1')
        expect(userContextsNames.get('1').value).toBe('server 2')
        expect(userContextsNames.get('2').value).toBe('server 1')

        const userContexts = component.state.group.get('userContexts.contexts') as UntypedFormArray
        expect(userContexts).toBeTruthy()
        expect(userContexts.length).toBe(3)
        expect(userContexts.get('0').value).toEqual({
            'subnet-name': 'server 1',
        })
        expect(userContexts.get('1').value).toEqual({
            'subnet-name': 'server 2',
        })
        expect(userContexts.get('2').value).toEqual({
            'subnet-name': 'server 1',
        })
    }))

    it('should revert the changes in the form', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseBeginSubnet4))
        component.subnetId = 123
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.state.group.get('selectedDaemonGroups').setValue([1])
        component.onDaemonGroupsChange({
            itemValue: 0,
        })
        tick()
        fixture.detectChanges()

        let options = component.state.group.get('options.data') as UntypedFormArray
        options.get('0.0.optionFields.0.control')?.setValue('192.0.2.3')

        let parameters = component.state.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        parameters.get('allocator.values.0')?.setValue('flq')

        let userContextsNames = component.state.group.get('userContexts.names') as UntypedFormArray
        userContextsNames.get('0')?.setValue('server 10')

        let userContexts = component.state.group.get('userContexts.contexts') as UntypedFormArray
        userContexts.get('0')?.setValue({
            'subnet-name': 'server 10',
            foo: 'bar',
        })

        component.onRevert()

        options = component.state.group.get('options.data') as UntypedFormArray
        expect(options.length).toBe(2)
        expect(options.get('0.0.optionFields.0.control')?.value).toBe('192.0.2.1')
        expect(options.get('1.0.optionFields.0.control')?.value).toBe('192.0.2.2')

        parameters = component.state.group.get('parameters') as FormGroup<KeaSubnetParametersForm>
        expect(parameters.get('allocator.unlocked')?.value).toBeTrue()
        expect((parameters.get('allocator.values') as UntypedFormArray).length).toBe(2)
        expect(parameters.get('allocator.values.0')?.value).toBe('random')
        expect(parameters.get('allocator.values.1')?.value).toBe('iterative')

        userContextsNames = component.state.group.get('userContexts.names') as UntypedFormArray
        expect(userContextsNames.length).toBe(2)
        expect(userContextsNames.get('0').value).toBe('server 1')
        expect(userContextsNames.get('1').value).toBe('server 2')

        userContexts = component.state.group.get('userContexts.contexts') as UntypedFormArray
        expect(userContexts.length).toBe(2)
        expect(userContexts.get('0').value).toEqual({
            'subnet-name': 'server 1',
        })
        expect(userContexts.get('1').value).toEqual({
            'subnet-name': 'server 2',
        })

        flush()
        // TODO: this test should be probably moved away from Karma tests. flush() is saving us from: Error: 11 timer(s) still in the queue.
    }))

    it('should add and remove the pool', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.onAddressPoolAdd()
        tick()
        fixture.detectChanges()

        const poolsPanel = fixture.debugElement.query(By.css('[legend="Pools"]'))
        expect(poolsPanel).toBeTruthy()

        // Expand the tab.
        const tabs = poolsPanel.queryAll(By.css('.p-accordionpanel'))
        expect(tabs.length).toBe(2)
        const link = tabs[1].query(By.css('.p-accordionheader'))
        expect(link).toBeTruthy()
        link.nativeElement.click()
        tick()
        fixture.detectChanges()

        expect(tabs[1].nativeElement.innerText).toContain('New Pool')
        const poolDeleteBtn = tabs[1].query(By.css('[label="Delete Pool"]'))
        expect(poolDeleteBtn).toBeTruthy()

        let pools = component.state.group.get('pools') as FormArray<FormGroup<AddressPoolForm>>
        expect(pools).toBeTruthy()
        expect(pools.length).toBe(2)

        spyOn(messageService, 'add').and.callThrough()
        component.onAddressPoolDelete(1)
        tick()
        fixture.detectChanges()
        pools = component.state.group.get('pools') as FormArray<FormGroup<AddressPoolForm>>
        expect(pools).toBeTruthy()
        expect(pools.length).toBe(1)
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should add and remove the prefix pool', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValue(wrapInHttpResponse(cannedResponseBeginSubnet6))
        component.subnetId = 234
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        component.onPrefixPoolAdd()
        tick()
        fixture.detectChanges()

        const poolsPanel = fixture.debugElement.query(By.css('[legend="Prefix Delegation Pools"]'))
        expect(poolsPanel).toBeTruthy()

        // Expand the tab.
        const tabs = poolsPanel.queryAll(By.css('.p-accordionpanel'))
        expect(tabs.length).toBe(2)
        const link = tabs[1].query(By.css('.p-accordionheader'))
        expect(link).toBeTruthy()
        link.nativeElement.click()
        tick()
        fixture.detectChanges()

        expect(tabs[1].nativeElement.innerText).toContain('New Pool')
        const poolDeleteBtn = tabs[1].query(By.css('[label="Delete Pool"]'))
        expect(poolDeleteBtn).toBeTruthy()

        let pools = component.state.group.get('prefixPools') as FormArray<FormGroup<PrefixPoolForm>>
        expect(pools).toBeTruthy()
        expect(pools.length).toBe(2)

        spyOn(messageService, 'add').and.callThrough()
        component.onPrefixPoolDelete(1)
        tick()
        fixture.detectChanges()

        pools = component.state.group.get('prefixPools') as FormArray<FormGroup<PrefixPoolForm>>
        expect(pools).toBeTruthy()
        expect(pools.length).toBe(1)
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should emit cancel event', () => {
        spyOn(component.formCancel, 'emit')
        component.onCancel()
        expect(component.formCancel.emit).toHaveBeenCalled()
    })

    it('should present an error message when begin transaction fails', fakeAsync(() => {
        spyOn(dhcpApi, 'updateSubnetBegin').and.returnValues(
            throwError({ status: 404 }),
            wrapInHttpResponse(cannedResponseBeginSubnet4)
        )
        component.subnetId = 123
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.state.initError).toEqual('status: 404')

        const messageElement = fixture.debugElement.query(By.css('p-message'))
        expect(messageElement).toBeTruthy()
        expect(messageElement.nativeElement.outerText).toContain(component.state.initError)

        const retryButton = fixture.debugElement.query(By.css('[label="Retry"]'))
        expect(retryButton).toBeTruthy()
        expect(retryButton.nativeElement.outerText).toBe('Retry')

        component.onRetry()
        tick()
        fixture.detectChanges()
        tick()

        expect(fixture.debugElement.query(By.css('p-message'))).toBeFalsy()
        expect(fixture.debugElement.query(By.css('[label="Retry"]'))).toBeFalsy()
        expect(fixture.debugElement.query(By.css('[label="Submit"]'))).toBeTruthy()
    }))
})
