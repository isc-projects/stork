import { ModifyDeep } from './utiltypes'

describe('util types', () => {
    it('MofifyDeep type assertions', () => {
        interface A {
            foo: string
            bar: string
            nested: {
                baz: string
                biz: string
            }
        }

        interface B {
            bar: number
            nested: {
                baz: number
            }
        }

        // Expect no typing errors
        const obj: ModifyDeep<A, B> = {
            foo: 'foo',
            bar: 1,
            nested: {
                baz: 2,
                biz: 'biz',
            },
        }
        expect(obj).toBeDefined()
    })
})
