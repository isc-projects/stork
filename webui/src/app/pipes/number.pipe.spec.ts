import { NumberPipe } from './number.pipe'

describe('NumberPipePipe', () => {
    it('creates an instance', () => {
        const pipe = new NumberPipe()
        expect(pipe).toBeTruthy()
    })

    it('returns null for a null value', () => {
        const pipe = new NumberPipe()
        expect(pipe.transform(null)).toBeNull()
        expect(pipe.transform(undefined)).toBeNull()
    })

    it('formats the numbers', () => {
        const pipe = new NumberPipe()
        expect(pipe.transform(42, 'en-US')).toBe('42')
        expect(pipe.transform(123456, 'en-US')).toBe('123,456')
    })

    it('converts the numeric strings', () => {
        const pipe = new NumberPipe()
        expect(pipe.transform('42', 'en-US')).toBe('42')
        expect(pipe.transform('123456', 'en-US')).toBe('123,456')
    })

    it('returns null for non-numeric strings', () => {
        const pipe = new NumberPipe()
        expect(pipe.transform('foo')).toBeNull()
    })
})
