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

    it('should display false for a boolean false value', () => {
        const pipe = new PlaceholderPipe()
        const placeholder = pipe.transform(false as any, 'foo', 'bar')
        expect(placeholder).toBe('false')
    })

    it('should display true for a boolean true value', () => {
        const pipe = new PlaceholderPipe()
        const placeholder = pipe.transform(true as any, 'foo', 'bar')
        expect(placeholder).toBe('true')
    })

    it('should display zero for a zero number', () => {
        const pipe = new PlaceholderPipe()
        const placeholder = pipe.transform(0 as any, 'foo', 'bar')
        expect(placeholder).toBe('0')
    })

    it('should display transformed array', () => {
        const pipe = new PlaceholderPipe()
        const placeholder = pipe.transform(['foo', 'bar'])
        expect(placeholder).toBe('foo,bar')
    })

    it('should display empty placeholder for empty array', () => {
        const pipe = new PlaceholderPipe()
        const placeholder = pipe.transform([])
        expect(placeholder).toBe('(empty)')
    })
})
