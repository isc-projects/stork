import { FormBuilder } from '@angular/forms'
import { StorkValidators } from './validators'

describe('StorkValidators', () => {
    let formBuilder: FormBuilder = new FormBuilder()

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

    it('validates fqdn', () => {
        expect(StorkValidators.fqdn(false)(formBuilder.control('a..bc'))).toBeTruthy()
        expect(StorkValidators.fqdn(false)(formBuilder.control('a.b'))).toBeTruthy()
        expect(StorkValidators.fqdn(false)(formBuilder.control('test-.xyz'))).toBeTruthy()
        expect(StorkValidators.fqdn(false)(formBuilder.control('-test.xyz'))).toBeTruthy()
        expect(StorkValidators.fqdn(false)(formBuilder.control('test.xyz.'))).toBeTruthy()
        expect(StorkValidators.fqdn(false)(formBuilder.control('.test.xyz'))).toBeTruthy()
        expect(StorkValidators.fqdn(false)(formBuilder.control('test'))).toBeTruthy()

        expect(StorkValidators.fqdn(false)(formBuilder.control('a.bc'))).toBeFalsy()
        expect(StorkValidators.fqdn(false)(formBuilder.control('test--abc.ec-a-b.xyz'))).toBeFalsy()
        expect(StorkValidators.fqdn(false)(formBuilder.control('test.abc.xyz'))).toBeFalsy()
        expect(StorkValidators.fqdn(false)(formBuilder.control('a123.xyz'))).toBeFalsy()
    })

    it('validates partial fqdn', () => {
        expect(StorkValidators.fqdn(true)(formBuilder.control('a..bc'))).toBeTruthy()
        expect(StorkValidators.fqdn(true)(formBuilder.control('test-.xyz'))).toBeTruthy()
        expect(StorkValidators.fqdn(true)(formBuilder.control('-test.xyz'))).toBeTruthy()
        expect(StorkValidators.fqdn(true)(formBuilder.control('test.xyz.'))).toBeTruthy()
        expect(StorkValidators.fqdn(true)(formBuilder.control('.test.xyz'))).toBeTruthy()

        expect(StorkValidators.fqdn(true)(formBuilder.control('a.bc'))).toBeFalsy()
        expect(StorkValidators.fqdn(true)(formBuilder.control('test--abc.x-yz'))).toBeFalsy()
        expect(StorkValidators.fqdn(true)(formBuilder.control('test.abc.xyz'))).toBeFalsy()
        expect(StorkValidators.fqdn(true)(formBuilder.control('test'))).toBeFalsy()
    })
})
