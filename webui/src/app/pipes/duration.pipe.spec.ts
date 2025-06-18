import { DurationPipe } from './duration.pipe'

describe('DurationPipe', () => {
    it('create an instance', () => {
        const pipe = new DurationPipe()
        expect(pipe).toBeTruthy()
    })

    it('should format duration correctly', () => {
        const pipe = new DurationPipe()
        expect(pipe.transform(null)).toBe(null)
        expect(pipe.transform(undefined)).toBe(undefined)
        expect(pipe.transform('1h2m3.4567s')).toBe('1 hour 2 minutes 3.5 seconds')
        expect(pipe.transform('1h2m3.1s')).toBe('1 hour 2 minutes 3.1 seconds')
        expect(pipe.transform('0s')).toBe('0 seconds')
        expect(pipe.transform('1m')).toBe('1 minute')
        expect(pipe.transform('2h3m4s')).toBe('2 hours 3 minutes 4 seconds')
        expect(pipe.transform('1h2m3.4567s')).toBe('1 hour 2 minutes 3.5 seconds')
        expect(pipe.transform('3d')).toBe('3 days')
        expect(pipe.transform('0d')).toBe('0 seconds')
        expect(pipe.transform('')).toBe('0 seconds')
        expect(pipe.transform('1w')).toBe('1 w')
        expect(pipe.transform('42ms')).toBe('42 milliseconds')
        expect(pipe.transform('42µs')).toBe('42 microseconds')
        expect(pipe.transform('42ns')).toBe('42 nanoseconds')
    })

    it('should format duration specified as a number correctly', () => {
        const pipe = new DurationPipe()
        expect(pipe.transform(42)).toBe('42 seconds')
        expect(pipe.transform(42.1)).toBe('42.1 seconds')
        expect(pipe.transform(100)).toBe('1 minute 40 seconds')
        expect(pipe.transform(3723)).toBe('1 hour 2 minutes 3 seconds')
    })

    it('should not crash if the value is invalid', () => {
        const pipe = new DurationPipe()
        expect(pipe.transform('invalid')).toBe('0 seconds')
        expect(pipe.transform('0')).toBe('0 seconds')
    })
})
