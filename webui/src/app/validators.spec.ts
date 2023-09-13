import { UntypedFormBuilder } from '@angular/forms'
import { StorkValidators } from './validators'

describe('StorkValidators', () => {
    let formBuilder: UntypedFormBuilder = new UntypedFormBuilder()

    it('validates hex identifier', () => {
        // Doesn't contain hexadecimal digits.
        expect(StorkValidators.hexIdentifier()(formBuilder.control('value'))).toBeTruthy()
        // Good identifier with colons as separator.
        expect(StorkValidators.hexIdentifier()(formBuilder.control('01:02:03'))).toBeFalsy()
        // Good identifier with dashes as a separator.
        expect(StorkValidators.hexIdentifier()(formBuilder.control('ca-fe-03'))).toBeFalsy()
        // Spaces as separator are not supported.
        expect(StorkValidators.hexIdentifier()(formBuilder.control('ca fe 03'))).toBeTruthy()
        // Empty string is fine for this validator.
        expect(StorkValidators.hexIdentifier()(formBuilder.control(''))).toBeFalsy()
    })

    it('validates hex identifier length', () => {
        expect(StorkValidators.hexIdentifierLength(6)(formBuilder.control('01:02:03'))).toBeFalsy()
        expect(StorkValidators.hexIdentifierLength(8)(formBuilder.control('112233'))).toBeFalsy()
        expect(StorkValidators.hexIdentifierLength(10)(formBuilder.control('ac-de-aa'))).toBeFalsy()

        expect(StorkValidators.hexIdentifierLength(4)(formBuilder.control('ab-cd-ef'))).toBeTruthy()
        expect(StorkValidators.hexIdentifierLength(2)(formBuilder.control('ee:02:90'))).toBeTruthy()
        expect(StorkValidators.hexIdentifierLength(6)(formBuilder.control('01020389'))).toBeTruthy()
    })

    it('validates IPv4 address', () => {
        // Partial address is not valid.
        expect(StorkValidators.ipv4()(formBuilder.control('10.0.'))).toBeTruthy()
        // Prefix is not valid.
        expect(StorkValidators.ipv4()(formBuilder.control('10.0.0.0/24'))).toBeTruthy()
        // Too many dots.
        expect(StorkValidators.ipv4()(formBuilder.control('192.0..2.1'))).toBeTruthy()
        // Dot after address.
        expect(StorkValidators.ipv4()(formBuilder.control('192.0.2.1.'))).toBeTruthy()
        // IPv6 address is not valid.
        expect(StorkValidators.ipv4()(formBuilder.control('2001:db8:1::1'))).toBeTruthy()
        // Must use dots to separate the IP address bytes.
        expect(StorkValidators.ipv4()(formBuilder.control('192x0x2x1'))).toBeTruthy()
        // Too high numbers.
        expect(StorkValidators.ipv4()(formBuilder.control('999.999.999.999'))).toBeTruthy()
        // Valid address.
        expect(StorkValidators.ipv4()(formBuilder.control('192.0.2.1'))).toBeFalsy()
    })

    it('validates IPv6 address', () => {
        // Partial address is not valid.
        expect(StorkValidators.ipv6()(formBuilder.control('2001'))).toBeTruthy()
        // Dots are not valid.
        expect(StorkValidators.ipv6()(formBuilder.control('3000..'))).toBeTruthy()
        // No colons at the end.
        expect(StorkValidators.ipv6()(formBuilder.control('3001:123'))).toBeTruthy()
        // Ipv6 address is not valid.
        expect(StorkValidators.ipv6()(formBuilder.control('192.0.2.1'))).toBeTruthy()
        // Valid address.
        expect(StorkValidators.ipv6()(formBuilder.control('3000:1:2::3'))).toBeFalsy()
    })

    it('validates IPv6 prefix', () => {
        // Partial prefix is not valid.
        expect(StorkValidators.ipv6Prefix()(formBuilder.control('2001::'))).toBeTruthy()
        // Dots are not valid.
        expect(StorkValidators.ipv6Prefix()(formBuilder.control('3000../64'))).toBeTruthy()
        // No colons at the end.
        expect(StorkValidators.ipv6Prefix()(formBuilder.control('3001:123/64'))).toBeTruthy()
        // IPv4 prefix is not valid.
        expect(StorkValidators.ipv6Prefix()(formBuilder.control('192.0.2.0/24'))).toBeTruthy()
        // Valid prefix.
        expect(StorkValidators.ipv6Prefix()(formBuilder.control('3000:1:2::/64'))).toBeFalsy()
    })

    it('validates full fqdn', () => {
        expect(StorkValidators.fullFqdn(formBuilder.control('a..bc.'))).toBeTruthy()
        expect(StorkValidators.fullFqdn(formBuilder.control('a.b.'))).toBeTruthy()
        expect(StorkValidators.fullFqdn(formBuilder.control('test-.xyz.'))).toBeTruthy()
        expect(StorkValidators.fullFqdn(formBuilder.control('-test.xyz.'))).toBeTruthy()
        expect(StorkValidators.fullFqdn(formBuilder.control('.test.xyz.'))).toBeTruthy()
        expect(StorkValidators.fullFqdn(formBuilder.control('test'))).toBeTruthy()
        expect(StorkValidators.fullFqdn(formBuilder.control('a.bc'))).toBeTruthy()

        expect(StorkValidators.fullFqdn(formBuilder.control('test--abc.ec-a-b.xyz.'))).toBeFalsy()
        expect(StorkValidators.fullFqdn(formBuilder.control('test.abc.xyz.'))).toBeFalsy()
        expect(StorkValidators.fullFqdn(formBuilder.control('a123.xyz.'))).toBeFalsy()
    })

    it('validates partial fqdn', () => {
        expect(StorkValidators.partialFqdn(formBuilder.control('a..bc'))).toBeTruthy()
        expect(StorkValidators.partialFqdn(formBuilder.control('test-.xyz'))).toBeTruthy()
        expect(StorkValidators.partialFqdn(formBuilder.control('-test.xyz'))).toBeTruthy()
        expect(StorkValidators.partialFqdn(formBuilder.control('test.xyz.'))).toBeTruthy()
        expect(StorkValidators.partialFqdn(formBuilder.control('.test.xyz'))).toBeTruthy()

        expect(StorkValidators.partialFqdn(formBuilder.control('a.bc'))).toBeFalsy()
        expect(StorkValidators.partialFqdn(formBuilder.control('test--abc.x-yz'))).toBeFalsy()
        expect(StorkValidators.partialFqdn(formBuilder.control('test.abc.xyz'))).toBeFalsy()
        expect(StorkValidators.partialFqdn(formBuilder.control('test'))).toBeFalsy()
    })

    it('validates fqdn', () => {
        // Invalid FQDN should cause an error.
        expect(StorkValidators.fqdn(formBuilder.control('test.'))).toBeTruthy()

        // A full or partial FQDN should be fine.
        expect(StorkValidators.fqdn(formBuilder.control('test--abc.ec-a-b.xyz.'))).toBeFalsy()
        expect(StorkValidators.fqdn(formBuilder.control('test'))).toBeFalsy()
    })
})
