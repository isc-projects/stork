import { UpdateSharedNetworkBeginResponse } from '../backend'
import { IPType } from '../iptype'
import { SharedNetworkFormState } from './shared-network-form'

describe('SharedNetworkFormState', () => {
    let response: UpdateSharedNetworkBeginResponse

    beforeEach(() => {
        response = {
            id: 5,
            sharedNetwork: {
                id: 123,
                name: 'stanza',
                localSharedNetworks: [
                    {
                        appId: 111,
                        daemonId: 1,
                        appName: 'server 1',
                    },
                    {
                        appId: 222,
                        daemonId: 2,
                        appName: 'server 2',
                    },
                ],
            },
            daemons: [
                {
                    id: 1,
                    name: 'dhcp4',
                    app: {
                        id: 1,
                        name: 'one',
                        type: 'kea',
                    },
                    version: '3.0.0',
                },
                {
                    id: 3,
                    name: 'dhcp6',
                    app: {
                        id: 3,
                        name: 'three',
                        type: 'kea',
                    },
                    version: '3.0.0',
                },
                {
                    id: 2,
                    name: 'dhcp4',
                    app: {
                        id: 2,
                        name: 'two',
                        type: 'kea',
                    },
                    version: '3.0.0',
                },
                {
                    id: 4,
                    name: 'dhcp6',
                    app: {
                        id: 4,
                        name: 'four',
                        type: 'kea',
                    },
                    version: '3.0.0',
                },
                {
                    id: 5,
                    name: 'dhcp6',
                    app: {
                        id: 5,
                        name: 'five',
                        type: 'kea',
                    },
                    version: '3.0.0',
                },
            ],
            sharedNetworks4: ['floor1', 'floor2', 'floor3'],
            sharedNetworks6: ['ground1', 'ground2'],
            clientClasses: ['foo', 'bar'],
        }
    })

    it('should create form state', () => {
        let state = new SharedNetworkFormState()
        expect(state.clientClasses).toEqual([])
        expect(state.dhcpv6).toBeFalse()
        expect(state.existingSharedNetworkNames).toEqual([])
        expect(state.filteredDaemons).toEqual([])
        expect(state.group).toBeFalsy()
        expect(state.initError).toBeFalsy()
        expect(state.ipType).toBe(IPType.IPv4)
        expect(state.loaded).toBeFalse()
        expect(state.savedSharedNetworkBeginData).toBeFalsy()
        expect(state.servers).toEqual([])
        expect(state.sharedNetworkId).toBe(0)
        expect(state.transactionId).toBe(0)
    })

    it('should initialize IPv4 form state', () => {
        let state = new SharedNetworkFormState()
        state.sharedNetworkId = 123
        state.initStateFromServerResponse(response)

        // Client classes.
        expect(state.clientClasses).toContain({ name: 'foo' })
        expect(state.clientClasses).toContain({ name: 'bar' })

        // Universe.
        expect(state.dhcpv6).toBeFalse()
        expect(state.ipType).toBe(IPType.IPv4)

        // Existing shared networks.
        expect(state.existingSharedNetworkNames).toEqual(['floor1', 'floor2', 'floor3'])

        // Filtered daemons.
        expect(state.filteredDaemons).toContain({
            id: 1,
            appId: 1,
            appType: 'kea',
            name: 'dhcp4',
            version: '3.0.0',
            label: 'one/dhcp4',
        })
        expect(state.filteredDaemons).toContain({
            id: 3,
            appId: 3,
            appType: 'kea',
            name: 'dhcp6',
            version: '3.0.0',
            label: 'three/dhcp6',
        })
        expect(state.filteredDaemons).toContain({
            id: 2,
            appId: 2,
            appType: 'kea',
            name: 'dhcp4',
            version: '3.0.0',
            label: 'two/dhcp4',
        })
        expect(state.filteredDaemons).toContain({
            id: 4,
            appId: 4,
            appType: 'kea',
            name: 'dhcp6',
            version: '3.0.0',
            label: 'four/dhcp6',
        })
        expect(state.filteredDaemons).toContain({
            id: 5,
            appId: 5,
            appType: 'kea',
            name: 'dhcp6',
            version: '3.0.0',
            label: 'five/dhcp6',
        })

        // Form group.
        expect(state.group).toBeFalsy()

        // Initialization error.
        expect(state.initError).toBeFalsy()

        // Other.
        expect(state.loaded).toBeFalse()
        expect(state.savedSharedNetworkBeginData).toBeFalsy()

        // Servers selection.
        expect(state.servers).toEqual(['one/dhcp4', 'two/dhcp4'])

        // Identifiers.
        expect(state.sharedNetworkId).toBe(123)
        expect(state.transactionId).toBe(5)
    })

    it('should initialize IPv6 form state', () => {
        let ipv6Response: UpdateSharedNetworkBeginResponse = {
            id: 3,
            sharedNetwork: {
                id: 234,
                name: 'stanza',
                universe: 6,
                localSharedNetworks: [
                    {
                        appId: 111,
                        daemonId: 3,
                        appName: 'server 1',
                    },
                    {
                        appId: 222,
                        daemonId: 4,
                        appName: 'server 2',
                    },
                ],
            },
            daemons: [
                {
                    id: 1,
                    name: 'dhcp4',
                    app: {
                        id: 1,
                        name: 'one',
                        type: 'kea',
                    },
                    version: '3.0.0',
                },
                {
                    id: 3,
                    name: 'dhcp6',
                    app: {
                        id: 3,
                        name: 'three',
                        type: 'kea',
                    },
                    version: '3.0.0',
                },
                {
                    id: 2,
                    name: 'dhcp4',
                    app: {
                        id: 2,
                        name: 'two',
                        type: 'kea',
                    },
                    version: '3.0.0',
                },
                {
                    id: 4,
                    name: 'dhcp6',
                    app: {
                        id: 4,
                        name: 'four',
                        type: 'kea',
                    },
                    version: '3.0.0',
                },
                {
                    id: 5,
                    name: 'dhcp6',
                    app: {
                        id: 5,
                        name: 'five',
                        type: 'kea',
                    },
                    version: '3.0.0',
                },
            ],
            sharedNetworks4: ['floor1', 'floor2', 'floor3'],
            sharedNetworks6: ['ground1', 'ground2'],
            clientClasses: ['foo', 'bar'],
        }
        let state = new SharedNetworkFormState()
        state.sharedNetworkId = 234
        state.initStateFromServerResponse(ipv6Response)

        // Client classes.
        expect(state.clientClasses).toContain({ name: 'foo' })
        expect(state.clientClasses).toContain({ name: 'bar' })

        // Universe.
        expect(state.dhcpv6).toBeTrue()
        expect(state.ipType).toBe(IPType.IPv6)

        // Existing shared networks.
        expect(state.existingSharedNetworkNames).toEqual(['ground1', 'ground2'])

        // Filtered daemons.
        expect(state.filteredDaemons).toContain({
            id: 1,
            appId: 1,
            appType: 'kea',
            name: 'dhcp4',
            version: '3.0.0',
            label: 'one/dhcp4',
        })
        expect(state.filteredDaemons).toContain({
            id: 3,
            appId: 3,
            appType: 'kea',
            name: 'dhcp6',
            version: '3.0.0',
            label: 'three/dhcp6',
        })
        expect(state.filteredDaemons).toContain({
            id: 2,
            appId: 2,
            appType: 'kea',
            name: 'dhcp4',
            version: '3.0.0',
            label: 'two/dhcp4',
        })
        expect(state.filteredDaemons).toContain({
            id: 4,
            appId: 4,
            appType: 'kea',
            name: 'dhcp6',
            version: '3.0.0',
            label: 'four/dhcp6',
        })
        expect(state.filteredDaemons).toContain({
            id: 5,
            appId: 5,
            appType: 'kea',
            name: 'dhcp6',
            version: '3.0.0',
            label: 'five/dhcp6',
        })

        // Form group.
        expect(state.group).toBeFalsy()

        // Initialization error.
        expect(state.initError).toBeFalsy()

        // Other.
        expect(state.loaded).toBeFalse()
        expect(state.savedSharedNetworkBeginData).toBeFalsy()

        // Servers selection.
        expect(state.servers).toEqual(['three/dhcp6', 'four/dhcp6'])

        // Identifiers.
        expect(state.sharedNetworkId).toBe(234)
        expect(state.transactionId).toBe(3)
    })

    it('should mark form loaded', () => {
        let state = new SharedNetworkFormState()
        expect(state.loaded).toBeFalse()
        state.markLoaded()
        expect(state.loaded).toBeTrue()
    })

    it('should update servers', () => {
        let state = new SharedNetworkFormState()
        state.sharedNetworkId = 123
        state.initStateFromServerResponse(response)
        expect(state.servers).toEqual(['one/dhcp4', 'two/dhcp4'])
        state.updateServers([2])
        expect(state.servers).toEqual(['two/dhcp4'])
    })

    it('should update universe for selected daemons', () => {
        let state = new SharedNetworkFormState()
        state.initStateFromServerResponse(response)
        expect(state.updateFormForSelectedDaemons([3, 4])).toBeTrue()
        expect(state.dhcpv6).toBeTrue()
    })

    it('should update filtered daemons for selected daemons', () => {
        let state = new SharedNetworkFormState()
        state.initStateFromServerResponse(response)
        expect(state.updateFormForSelectedDaemons([3, 4])).toBeTrue()
        expect(state.filteredDaemons.length).toBe(3)
        expect(state.filteredDaemons).toContain({
            id: 3,
            appId: 3,
            appType: 'kea',
            name: 'dhcp6',
            version: '3.0.0',
            label: 'three/dhcp6',
        })
        expect(state.filteredDaemons).toContain({
            id: 4,
            appId: 4,
            appType: 'kea',
            name: 'dhcp6',
            version: '3.0.0',
            label: 'four/dhcp6',
        })
        expect(state.filteredDaemons).toContain({
            id: 5,
            appId: 5,
            appType: 'kea',
            name: 'dhcp6',
            version: '3.0.0',
            label: 'five/dhcp6',
        })
    })
})
