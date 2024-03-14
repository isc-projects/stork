import { HostsFilter, getBooleanFilterKeys, getNumericFilterKeys } from './hosts-filter'

describe('QueryParamsFilter', () => {
    let filter: Required<HostsFilter>

    beforeEach(() => {
        filter = {
            appId: 0,
            conflict: false,
            isGlobal: false,
            keaSubnetId: 0,
            migrationError: false,
            subnetId: 0,
            text: '',
        }
    })

    it('should return the boolean keys', () => {
        // Act
        const keys = getBooleanFilterKeys()

        // Assert
        // Check if all keys refer to boolean values.
        for (let key of keys) {
            expect(typeof filter[key]).toBe('boolean')
        }
        // Check if all boolean keys are listed.
        for (let key of Object.keys(filter)) {
            if (typeof filter[key] === 'boolean') {
                expect(keys).toContain(key as keyof HostsFilter)
            }
        }
    })

    it('should return the numeric keys', () => {
        // Act
        const keys = getNumericFilterKeys()

        // Assert
        // Check if all keys refer to numeric values.
        for (let key of keys) {
            expect(typeof filter[key]).toBe('number')
        }
        // Check if all numeric keys are listed.
        for (let key of Object.keys(filter)) {
            if (typeof filter[key] === 'number') {
                expect(keys).toContain(key as keyof HostsFilter)
            }
        }
    })

    it('should not have unlisted keys', () => {
        // Act
        const keys = ['text', ...getNumericFilterKeys(), ...getBooleanFilterKeys()].sort()

        // Assert
        expect(keys).toEqual(Object.keys(filter).sort())
    })
})
