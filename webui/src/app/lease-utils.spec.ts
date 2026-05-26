import { stateToString } from './lease-utils'

describe('Lease Utils', () => {
    it('should return correct lease state name', () => {
        expect(stateToString(null)).toBe('Valid')
        expect(stateToString(0)).toBe('Valid')
        expect(stateToString(1)).toBe('Declined')
        expect(stateToString(2)).toBe('Expired/Reclaimed')
        expect(stateToString(3)).toBe('Released')
        expect(stateToString(4)).toBe('Registered')
        expect(stateToString(5)).toBe('(invalid state)')
    })
})
