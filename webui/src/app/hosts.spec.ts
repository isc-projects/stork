import { TestBed } from '@angular/core/testing'
import {
    hasDifferentLocalHostBootFields,
    hasDifferentLocalHostClientClasses,
    hasDifferentLocalHostData,
    hasDifferentLocalHostHostname,
    hasDifferentLocalHostIPReservations,
    hasDifferentLocalHostOptions,
} from './hosts'
import { LocalHost } from './backend'

describe('hosts', () => {
    beforeEach(() => TestBed.configureTestingModule({}))

    it('detects differences between DHCP options', () => {
        const localHosts: LocalHost[] = [
            {
                optionsHash: '123',
                clientClasses: ['foo', 'bar'],
            },
            {
                optionsHash: '234',
                clientClasses: ['foo', 'bar'],
            },
            {
                optionsHash: '123',
                clientClasses: ['foo', 'bar'],
            },
        ]
        expect(hasDifferentLocalHostOptions(localHosts)).toBeTrue()
        expect(hasDifferentLocalHostClientClasses(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostData(localHosts)).toBeTrue()
    })

    it('detects differences between client classes', () => {
        const localHosts = [
            {
                optionsHash: '123',
                clientClasses: ['foo', 'bar'],
            },
            {
                optionsHash: '123',
                clientClasses: ['foo'],
            },
            {
                optionsHash: '123',
                clientClasses: ['foo', 'bar'],
            },
        ]
        expect(hasDifferentLocalHostData(localHosts)).toBeTrue()
    })

    it('detects differences between boot fields', () => {
        const localHosts = [
            {
                optionsHash: '123',
                clientClasses: ['foo', 'bar'],
                nextServer: '192.0.2.1',
            },
            {
                optionsHash: '123',
                clientClasses: ['foo', 'bar'],
                nextServer: '192.0.2.2',
            },
            {
                optionsHash: '123',
                clientClasses: ['foo', 'bar'],
                nextServer: '192.0.2.1',
            },
        ]
        expect(hasDifferentLocalHostBootFields(localHosts)).toBeTrue()
        expect(hasDifferentLocalHostData(localHosts)).toBeTrue()
    })

    it('detects that there are no differences', () => {
        const localHosts = [
            {
                optionsHash: '123',
                clientClasses: ['foo', 'bar'],
                nextServer: '192.0.2.1',
                serverHostname: 'my-server',
                bootFileName: '/tmp/boot',
            },
            {
                optionsHash: '123',
                clientClasses: ['foo', 'bar'],
                nextServer: '192.0.2.1',
                serverHostname: 'my-server',
                bootFileName: '/tmp/boot',
            },
            {
                optionsHash: '123',
                clientClasses: ['foo', 'bar'],
                nextServer: '192.0.2.1',
                serverHostname: 'my-server',
                bootFileName: '/tmp/boot',
            },
        ]
        expect(hasDifferentLocalHostOptions(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostClientClasses(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostBootFields(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostData(localHosts)).toBeFalse()
    })

    it('detects no differences for all null options hashes', () => {
        const localHosts = [
            {
                optionsHash: null,
                clientClasses: ['foo', 'bar'],
            },
            {
                optionsHash: null,
                clientClasses: ['foo', 'bar'],
            },
            {
                optionsHash: null,
                clientClasses: ['foo', 'bar'],
            },
        ]
        expect(hasDifferentLocalHostOptions(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostClientClasses(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostData(localHosts)).toBeFalse()
    })

    it('detects no differences when client classes are in different order', () => {
        const localHosts = [
            {
                optionsHash: null,
                clientClasses: ['foo', 'bar', 'baz'],
            },
            {
                optionsHash: null,
                clientClasses: ['foo', 'baz', 'bar'],
            },
            {
                optionsHash: null,
                clientClasses: ['baz', 'bar', 'foo'],
            },
        ]
        expect(hasDifferentLocalHostOptions(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostClientClasses(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostData(localHosts)).toBeFalse()
    })

    it('detects no differences for all null client', () => {
        const localHosts = [
            {
                clientClasses: null,
            },
            {
                clientClasses: null,
            },
            {
                clientClasses: null,
            },
        ]
        expect(hasDifferentLocalHostOptions(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostClientClasses(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostData(localHosts)).toBeFalse()
    })

    it('detects differences for some null client classes', () => {
        const localHosts = [
            {
                clientClasses: null,
            },
            {
                clientClasses: ['foo'],
            },
            {
                clientClasses: ['foo'],
            },
        ]
        expect(hasDifferentLocalHostOptions(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostClientClasses(localHosts)).toBeTrue()
        expect(hasDifferentLocalHostData(localHosts)).toBeTrue()
    })

    it('detects differences for next server', () => {
        const localHosts = [
            {
                nextServer: '192.0.2.2',
                serverHostname: 'foo',
                bootFileName: '/tmp/bootfile',
            },
            {
                nextServer: '192.0.2.1',
                serverHostname: 'foo',
                bootFileName: '/tmp/bootfile',
            },
            {
                nextServer: '192.0.2.1',
                serverHostname: 'foo',
                bootFileName: '/tmp/bootfile',
            },
        ]
        expect(hasDifferentLocalHostBootFields(localHosts)).toBeTrue()
    })

    it('detects differences for server hostname', () => {
        const localHosts = [
            {
                nextServer: '192.0.2.1',
                serverHostname: 'foo',
                bootFileName: '/tmp/bootfile',
            },
            {
                nextServer: '192.0.2.1',
                serverHostname: 'bar',
                bootFileName: '/tmp/bootfile',
            },
            {
                nextServer: '192.0.2.1',
                serverHostname: 'foo',
                bootFileName: '/tmp/bootfile',
            },
        ]
        expect(hasDifferentLocalHostBootFields(localHosts)).toBeTrue()
    })

    it('detects differences for boot file name', () => {
        const localHosts = [
            {
                nextServer: '192.0.2.1',
                serverHostname: 'foo',
                bootFileName: '/tmp/bootfile',
            },
            {
                nextServer: '192.0.2.1',
                serverHostname: 'foo',
                bootFileName: '/tmp/bootx',
            },
            {
                nextServer: '192.0.2.1',
                serverHostname: 'foo',
                bootFileName: '/tmp/bootfile',
            },
        ]
        expect(hasDifferentLocalHostBootFields(localHosts)).toBeTrue()
    })

    it('detects differences between IP reservations', () => {
        const localHosts: Partial<LocalHost>[] = [
            {
                ipReservations: [
                    {
                        address: '10.0.0.1',
                    },
                ],
            },
            {
                ipReservations: [
                    {
                        address: '10.0.0.1',
                    },
                ],
            },
            {
                ipReservations: [
                    {
                        address: '10.1.1.1',
                    },
                ],
            },
        ]

        expect(hasDifferentLocalHostIPReservations(localHosts as LocalHost[])).toBeTrue()
    })

    it('detects differences between hostnames', () => {
        const localHosts: Partial<LocalHost>[] = [
            {
                hostname: 'foo',
            },
            {
                hostname: 'bar',
            },
            {
                hostname: 'foo',
            },
        ]

        expect(hasDifferentLocalHostHostname(localHosts as LocalHost[])).toBeTrue()
    })

    it('detects no differences when there is a single local host', () => {
        const localHosts = [
            {
                optionsHash: '123',
                clientClasses: ['foo'],
                nextServer: '192.0.2.1',
            },
        ]
        expect(hasDifferentLocalHostClientClasses(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostBootFields(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostData(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostOptions(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostHostname(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostIPReservations(localHosts)).toBeFalse()
    })

    it('detects no differences when there is no local host', () => {
        const localHosts = []
        expect(hasDifferentLocalHostClientClasses(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostBootFields(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostData(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostOptions(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostHostname(localHosts)).toBeFalse()
        expect(hasDifferentLocalHostIPReservations(localHosts)).toBeFalse()
    })
})
