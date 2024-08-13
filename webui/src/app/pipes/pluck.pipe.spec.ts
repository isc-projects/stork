import { PluckPipe } from './pluck.pipe'

describe('PluckPipe', () => {
    it('create an instance', () => {
        const pipe = new PluckPipe()
        expect(pipe).toBeTruthy()
    })

    it('should return an array of values for a given key', () => {
        const pipe = new PluckPipe()
        const input = [
            { id: 1, name: 'John' },
            { id: 2, name: 'Doe' },
        ]
        const key = 'name'
        const output = ['John', 'Doe']
        expect(pipe.transform(input, key)).toEqual(output)
    })

    it('should return an array of undefined values for a non-existing key', () => {
        const pipe = new PluckPipe()
        const input = [
            { id: 1, name: 'John' },
            { id: 2, name: 'Doe' },
        ]
        const key = 'surname'
        const output = [undefined, undefined]
        expect(pipe.transform(input as any, key)).toEqual(output)
    })

    it('should return an empty array for an empty input', () => {
        const pipe = new PluckPipe()
        const input = []
        const key = 'name'
        const output = []
        expect(pipe.transform(input as any, key)).toEqual(output)
    })

    it('should return an empty for a non-array input', () => {
        const pipe = new PluckPipe()
        const input = 'not an array'
        const key = 'name'
        const output = []
        expect(pipe.transform(input as any, key)).toEqual(output)
    })
})
