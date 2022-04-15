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
})
