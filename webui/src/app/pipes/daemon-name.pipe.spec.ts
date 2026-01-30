import { DaemonNiceNamePipe } from './daemon-name.pipe'

describe('DaemonNiceNamePipe', () => {
    it('create an instance', () => {
        const pipe = new DaemonNiceNamePipe()
        expect(pipe).toBeTruthy()
    })

    it('should transform to nice name correctly', () => {
        const pipe = new DaemonNiceNamePipe()
        expect(pipe.transform(null)).toBe(null)
        expect(pipe.transform(undefined)).toBe(undefined)
        expect(pipe.transform('dhcp4')).toBe('DHCPv4')
        expect(pipe.transform('dhcp6')).toBe('DHCPv6')
        expect(pipe.transform('d2')).toBe('DDNS')
        expect(pipe.transform('ca')).toBe('CA')
        expect(pipe.transform('netconf')).toBe('NETCONF')
        expect(pipe.transform('named')).toBe('named')
        expect(pipe.transform('pdns')).toBe('pdns_server')
        expect(pipe.transform('unsupported')).toBe('Unsupported')
    })
})
