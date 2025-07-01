import { UnrootPipe } from './unroot.pipe'

describe('UnrootPipe', () => {
    it('create an instance', () => {
        const pipe = new UnrootPipe()
        expect(pipe).toBeTruthy()
    })

    it('should ignore null', () => {
        const pipe = new UnrootPipe()
        const result = pipe.transform(null)
        expect(result).toBe('(root)')
    })

    it('should ignore undefined', () => {
        const pipe = new UnrootPipe()
        const result = pipe.transform(undefined)
        expect(result).toBe('(root)')
    })

    it('should unroot regular name', () => {
        const pipe = new UnrootPipe()
        const result = pipe.transform('example.com')
        expect(result).toBe('example.com')
    })

    it('should unroot root name', () => {
        const pipe = new UnrootPipe()
        const result = pipe.transform(' . ')
        expect(result).toBe('(root)')
    })
})
