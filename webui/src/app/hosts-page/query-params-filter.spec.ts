import {
    QueryParamsFilter,
    getBooleanQueryParamsFilterKeys,
    getNumericQueryParamsFilterKeys,
} from './query-params-filter'

describe('QueryParamsFilter', () => {
    let filter: Required<QueryParamsFilter>

    beforeEach(() => {
        filter = {
            appId: 0,
            conflict: false,
            global: false,
            keaSubnetId: 0,
            migrationError: false,
            subnetId: 0,
            text: '',
        }
    })

    it('should return the boolean keys', () => {
        // Act
        const keys = getBooleanQueryParamsFilterKeys()

        // Assert
        // Check if all keys refer to boolean values.
        for (let key of keys) {
            expect(typeof filter[key]).toBe('boolean')
        }
        // Check if all boolean keys are listed.
        for (let key of Object.keys(filter)) {
            if (typeof filter[key] === 'boolean') {
                expect(keys).toContain(key as keyof QueryParamsFilter)
            }
        }
    })

    it('should return the numeric keys', () => {
        // Act
        const keys = getNumericQueryParamsFilterKeys()

        // Assert
        // Check if all keys refer to numeric values.
        for (let key of keys) {
            expect(typeof filter[key]).toBe('number')
        }
        // Check if all numeric keys are listed.
        for (let key of Object.keys(filter)) {
            if (typeof filter[key] === 'number') {
                expect(keys).toContain(key as keyof QueryParamsFilter)
            }
        }
    })

    it('should not have unlisted keys', () => {
        // Act
        const keys = ['text', ...getNumericQueryParamsFilterKeys(), ...getBooleanQueryParamsFilterKeys()].sort()

        // Assert
        expect(keys).toEqual(Object.keys(filter).sort())
    })
})
