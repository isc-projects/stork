import { HumanCountPipe } from './human-count.pipe'

describe('HumanCountPipe', () => {
    it('create an instance', () => {
        const pipe = new HumanCountPipe()
        expect(pipe).toBeTruthy()
    })

    it('human count should return proper string for any number', () => {
        // Arrange
        const pipe = new HumanCountPipe()
        const humanCount = pipe.transform

        const int = 12345678
        const float = 1234567890
        const bigInt = BigInt('1234567890123000000000000000000000')
        const str = 'foo'
        const nan = Number.NaN
        const boolean = true as any
        const nullValue = null

        // Act
        const strInt = humanCount(int)
        const strFloat = humanCount(float)
        const strBigInt = humanCount(bigInt)
        const strStr = humanCount(str)
        const nanStr = humanCount(nan)
        const boolStr = humanCount(boolean)
        const nullStr = humanCount(nullValue)

        // Assert
        expect(strInt).toBe('12.3M')
        expect(strFloat).toBe('1.2G')
        expect(strBigInt).toBe('1234567890.1Y')
        expect(strStr).toBe('foo')
        expect(nanStr).toBe('NaN')
        expect(boolStr).toBe('true')
        expect(nullStr).toBe('null')
    })

    it('should convert strings to numbers', () => {
        // Arrange
        const pipe = new HumanCountPipe()
        const humanCount = pipe.transform

        const int = '12345678'
        const bigInt = '1234567890000000000000000000000000'

        // Act
        const strInt = humanCount(int)
        const strBigInt = humanCount(bigInt)

        // Assert
        expect(strInt).toBe('12.3M')
        expect(strBigInt).toBe('1234567890.0Y')
    })
})
