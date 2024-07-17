import { UnhyphenPipe } from './unhyphen.pipe'

describe('UnhyphenPipe', () => {
    it('create an instance', () => {
        const pipe = new UnhyphenPipe()
        expect(pipe).toBeTruthy()
    })

    it('should ignore null', () => {
        const pipe = new UnhyphenPipe()
        const result = pipe.transform(null)
        expect(result).toBe(null)
    })

    it('should unhyphen regular key', () => {
        const pipe = new UnhyphenPipe()
        const result = pipe.transform('horse-power')
        expect(result).toBe('horsePower')
    })
})
