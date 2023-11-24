import { TestBed } from '@angular/core/testing'
import { AddressRange } from './address-range'

describe('AddressRange', () => {
    beforeEach(() => TestBed.configureTestingModule({}))

    it('should parse an IPv4 address range', () => {
        const range = AddressRange.fromStringRange('192.0.2.1 - 192.0.2.10')
        expect(range.getFirst()).toBe('192.0.2.1')
        expect(range.getLast()).toBe('192.0.2.10')
    })

    it('should parse an IPv4 address range without spaces', () => {
        const range = AddressRange.fromStringRange('192.0.2.1-192.0.2.10')
        expect(range.getFirst()).toBe('192.0.2.1')
        expect(range.getLast()).toBe('192.0.2.10')
    })

    it('should parse an IPv4 address range boundaries', () => {
        const range = AddressRange.fromStringBounds('192.0.2.1', '192.0.2.10')
        expect(range.getFirst()).toBe('192.0.2.1')
        expect(range.getLast()).toBe('192.0.2.10')
    })

    it('should parse an IPv6 address range', () => {
        const range = AddressRange.fromStringRange('3001:2:1:: - 3001:2:1:100::')
        expect(range.getFirst()).toBe('3001:2:1::')
        expect(range.getLast()).toBe('3001:2:1:100::')
    })

    it('should parse an IPv6 address range boundaries', () => {
        const range = AddressRange.fromStringBounds('3001:2:1::', '3001:2:1:100::')
        expect(range.getFirst()).toBe('3001:2:1::')
        expect(range.getLast()).toBe('3001:2:1:100::')
    })

    it('should throw for inconsistent family', () => {
        expect(() => AddressRange.fromStringRange('192.0.2.1 - 3001:2:1:100::')).toThrowError(
            'inconsistent IP address family in the 192.0.2.1 - 3001:2:1:100:: range'
        )
    })

    it('should throw for inconsistent family boundaries', () => {
        expect(() => AddressRange.fromStringBounds('192.0.2.1', '3001:2:1:100::')).toThrowError(
            'inconsistent IP address family in the 192.0.2.1-3001:2:1:100:: range'
        )
    })

    it('should throw for parsing invalid lower bound IP address', () => {
        expect(() => AddressRange.fromStringRange('192.0.2.-192.0.2.10')).toThrowError(
            'invalid IP addresses in the 192.0.2.-192.0.2.10 range'
        )
    })

    it('should throw for invalid lower bound IP address', () => {
        expect(() => AddressRange.fromStringBounds('192.0.2.', '192.0.2.10')).toThrowError(
            'invalid IP addresses in the 192.0.2.-192.0.2.10 range'
        )
    })

    it('should throw for parsing invalid upper bound IP address', () => {
        expect(() => AddressRange.fromStringRange('192.0.2.1-192.02.10')).toThrowError(
            'invalid IP addresses in the 192.0.2.1-192.02.10 range'
        )
    })

    it('should throw for invalid upper bound IP address', () => {
        expect(() => AddressRange.fromStringBounds('192.0.2.1', '192.02.10')).toThrowError(
            'invalid IP addresses in the 192.0.2.1-192.02.10 range'
        )
    })
})
