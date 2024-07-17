import { UncamelPipe } from './uncamel.pipe'

describe('UncamelPipe', () => {
    it('create an instance', () => {
        const pipe = new UncamelPipe()
        expect(pipe).toBeTruthy()
    })

    it('should ignore null', () => {
        const pipe = new UncamelPipe()
        const result = pipe.transform(null)
        expect(result).toBe(null)
    })

    it('should uncamel case regular key', () => {
        const pipe = new UncamelPipe()
        const result = pipe.transform('ddnsAttempt')
        expect(result).toBe('DDNS Attempt')
    })
})
