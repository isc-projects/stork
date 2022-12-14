import { TestBed } from '@angular/core/testing'
import { Host } from './backend/model/host'
import {
    hasDifferentLocalHostBootFields,
    hasDifferentLocalHostClientClasses,
    hasDifferentLocalHostData,
    hasDifferentLocalHostOptions,
} from './hosts'

describe('hosts', () => {
    beforeEach(() => TestBed.configureTestingModule({}))

    it('detects differences between DHCP options', () => {
        const host: Host = {
            localHosts: [
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
            ],
        }
        expect(hasDifferentLocalHostOptions(host)).toBeTrue()
        expect(hasDifferentLocalHostClientClasses(host)).toBeFalse()
        expect(hasDifferentLocalHostData(host)).toBeTrue()
    })

    it('detects differences between client classes', () => {
        const host: Host = {
            localHosts: [
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
            ],
        }
        expect(hasDifferentLocalHostData(host)).toBeTrue()
    })

    it('detects differences between boot fields', () => {
        const host: Host = {
            localHosts: [
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
            ],
        }
        expect(hasDifferentLocalHostBootFields(host)).toBeTrue()
        expect(hasDifferentLocalHostData(host)).toBeTrue()
    })

    it('detects that there are no differences', () => {
        const host: Host = {
            localHosts: [
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
            ],
        }
        expect(hasDifferentLocalHostOptions(host)).toBeFalse()
        expect(hasDifferentLocalHostClientClasses(host)).toBeFalse()
        expect(hasDifferentLocalHostBootFields(host)).toBeFalse()
        expect(hasDifferentLocalHostData(host)).toBeFalse()
    })

    it('detects no differences for all null options hashes', () => {
        const host: Host = {
            localHosts: [
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
            ],
        }
        expect(hasDifferentLocalHostOptions(host)).toBeFalse()
        expect(hasDifferentLocalHostClientClasses(host)).toBeFalse()
        expect(hasDifferentLocalHostData(host)).toBeFalse()
    })

    it('detects no differences when client classes are in different order', () => {
        const host: Host = {
            localHosts: [
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
            ],
        }
        expect(hasDifferentLocalHostOptions(host)).toBeFalse()
        expect(hasDifferentLocalHostClientClasses(host)).toBeFalse()
        expect(hasDifferentLocalHostData(host)).toBeFalse()
    })

    it('detects no differences for all null client', () => {
        const host: Host = {
            localHosts: [
                {
                    clientClasses: null,
                },
                {
                    clientClasses: null,
                },
                {
                    clientClasses: null,
                },
            ],
        }
        expect(hasDifferentLocalHostOptions(host)).toBeFalse()
        expect(hasDifferentLocalHostClientClasses(host)).toBeFalse()
        expect(hasDifferentLocalHostData(host)).toBeFalse()
    })

    it('detects differences for some null client classes', () => {
        const host: Host = {
            localHosts: [
                {
                    clientClasses: null,
                },
                {
                    clientClasses: ['foo'],
                },
                {
                    clientClasses: ['foo'],
                },
            ],
        }
        expect(hasDifferentLocalHostOptions(host)).toBeFalse()
        expect(hasDifferentLocalHostClientClasses(host)).toBeTrue()
        expect(hasDifferentLocalHostData(host)).toBeTrue()
    })

    it('detects differences for next server', () => {
        const host: Host = {
            localHosts: [
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
            ],
        }
        expect(hasDifferentLocalHostBootFields(host)).toBeTrue()
    })

    it('detects differences for server hostname', () => {
        const host: Host = {
            localHosts: [
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
            ],
        }
        expect(hasDifferentLocalHostBootFields(host)).toBeTrue()
    })

    it('detects differences for boot file name', () => {
        const host: Host = {
            localHosts: [
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
            ],
        }
        expect(hasDifferentLocalHostBootFields(host)).toBeTrue()
    })

    it('detects no differences when there is a single local host', () => {
        const host: Host = {
            localHosts: [
                {
                    optionsHash: '123',
                    clientClasses: ['foo'],
                    nextServer: '192.0.2.1',
                },
            ],
        }
        expect(hasDifferentLocalHostOptions(host)).toBeFalse()
        expect(hasDifferentLocalHostClientClasses(host)).toBeFalse()
        expect(hasDifferentLocalHostBootFields(host)).toBeFalse()
        expect(hasDifferentLocalHostData(host)).toBeFalse()
    })

    it('detects no differences when there is no local host', () => {
        const host: Host = {
            localHosts: [],
        }
        expect(hasDifferentLocalHostClientClasses(host)).toBeFalse()
        expect(hasDifferentLocalHostClientClasses(host)).toBeFalse()
        expect(hasDifferentLocalHostBootFields(host)).toBeFalse()
        expect(hasDifferentLocalHostData(host)).toBeFalse()
    })
})
