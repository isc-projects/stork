import { SurroundPipe } from './surround.pipe'

describe('SurroundPipe', () => {
    it('create an instance', () => {
        const pipe = new SurroundPipe()
        expect(pipe).toBeTruthy()
    })

    it('should surround a string', () => {
        const pipe = new SurroundPipe()
        const result = pipe.transform('2', '1', '3')
        expect(result).toBe('123')
    })

    it('should not surround null or undefined', () => {
        const pipe = new SurroundPipe()
        expect(pipe.transform(null, '1', '3')).toBe(null)
        expect(pipe.transform(undefined, '1', '3')).toBe(undefined)
    })
})
