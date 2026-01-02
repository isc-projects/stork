import { DNSZoneType } from '../backend'
import { ZoneTypeAliasPipe } from './zone-type-alias.pipe'

describe('ZoneTypeAliasPipe', () => {
    it('create an instance', () => {
        const pipe = new ZoneTypeAliasPipe()
        expect(pipe).toBeTruthy()
    })

    it('should ignore null', () => {
        const pipe = new ZoneTypeAliasPipe()
        const result = pipe.transform(null)
        expect(result).toBeNull()
    })

    it('should ignore undefined', () => {
        const pipe = new ZoneTypeAliasPipe()
        const result = pipe.transform(undefined)
        expect(result).toBeUndefined()
    })

    it('should transform master to primary', () => {
        const pipe = new ZoneTypeAliasPipe()
        const result = pipe.transform('master')
        expect(result).toBe('primary')
    })

    it('should transform slave to secondary', () => {
        const pipe = new ZoneTypeAliasPipe()
        const result = pipe.transform('slave')
        expect(result).toBe('secondary')
    })

    it('should return the original value if it is not master or slave', () => {
        const pipe = new ZoneTypeAliasPipe()
        for (const zoneType of Object.values(DNSZoneType)) {
            const result = pipe.transform(zoneType)
            expect(result).toBe(zoneType)
        }
    })
})
