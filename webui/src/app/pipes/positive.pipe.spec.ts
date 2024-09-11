import { PositivePipe } from './positive.pipe'

describe('PositivePipe', () => {
    it('create an instance', () => {
        const pipe = new PositivePipe()
        expect(pipe).toBeTruthy()
    })

    it('should zero negative bigint', () => {
        const pipe = new PositivePipe()
        const result = pipe.transform(BigInt(-5))
        expect(result).toBe(0)
    })

    it('should not zero positive bigint', () => {
        const pipe = new PositivePipe()
        const result = pipe.transform(BigInt(90))
        expect(result).toBe(BigInt(90))
    })

    it('should zero negative number', () => {
        const pipe = new PositivePipe()
        const result = pipe.transform(-9)
        expect(result).toBe(0)
    })

    it('should not zero positive number', () => {
        const pipe = new PositivePipe()
        const result = pipe.transform(89)
        expect(result).toBe(89)
    })

    it('should return null', () => {
        const pipe = new PositivePipe()
        const result = pipe.transform(null)
        expect(result).toBeNull()
    })

    it('should return undefined', () => {
        const pipe = new PositivePipe()
        const result = pipe.transform(undefined)
        expect(result).toBeUndefined()
    })
})
