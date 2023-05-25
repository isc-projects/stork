import { PlaceholderPipe } from './placeholder.pipe'

describe('PlaceholderPipe', () => {
    it('create an instance', () => {
        const pipe = new PlaceholderPipe()
        expect(pipe).toBeTruthy()
    })

    it('should display placeholder if value is empty', () => {
        const pipe = new PlaceholderPipe()
        const placeholder = pipe.transform('', 'foo', 'bar')
        expect(placeholder).toBe('bar')
    })

    it('should display placeholder if value is null', () => {
        const pipe = new PlaceholderPipe()
        const placeholder = pipe.transform(null, 'foo', 'bar')
        expect(placeholder).toBe('foo')
    })

    it('should display placeholder if value is undefined', () => {
        const pipe = new PlaceholderPipe()
        const placeholder = pipe.transform(undefined, 'foo', 'bar')
        expect(placeholder).toBe('foo')
    })

    it('should display value if value is provided', () => {
        const pipe = new PlaceholderPipe()
        const placeholder = pipe.transform('baz', 'foo', 'bar')
        expect(placeholder).toBe('baz')
    })
})
