import { clamp, humanCount } from './utils'

describe('utils', () => {
    it('clamps should return return proper number', () => {
        // Integers - in range
        expect(clamp(1, 0, 2)).toBe(1)
        // Integers - on lower bound
        expect(clamp(0, 0, 2)).toBe(0)
        // Integers - below lower bound
        expect(clamp(-1, 0, 2)).toBe(0)
        // Integers - on upper bound
        expect(clamp(2, 0, 2)).toBe(2)
        // Integers - above upper bound
        expect(clamp(3, 0, 2)).toBe(2)

        // Floats - in range
        expect(clamp(1.5, 0.4, 2.22)).toBe(1.5)
        // Floats - on lower bound
        expect(clamp(0.4, 0.4, 2.22)).toBe(0.4)
        // Floats - below lower bound
        expect(clamp(0.1, 0.4, 2.22)).toBe(0.4)
        // Floats - on upper bound
        expect(clamp(2.22, 0.4, 2.22)).toBe(2.22)
        // Floats - above upper bound
        expect(clamp(3.22, 0.4, 2.22)).toBe(2.22)

        // Floats - value as negative infinity
        expect(clamp(Number.NEGATIVE_INFINITY, 0, 1)).toBe(0)
        // Floats - value as positive infinity
        expect(clamp(Number.POSITIVE_INFINITY, 0, 1)).toBe(1)
        // Floats - bounds as infinities
        expect(clamp(3, Number.NEGATIVE_INFINITY, Number.POSITIVE_INFINITY)).toBe(3)
    })

    it('human count should return proper string for any number', () => {
        // Arrange
        const int = 12345678
        const float = 1234567890
        const bigInt = BigInt("1234567890000000000000000000000000")

        // Act
        const strInt = humanCount(int)
        const strFloat = humanCount(float)
        const strBigInt = humanCount(bigInt)

        // Assert
        expect(strInt).toBe("12.3 M")
        expect(strFloat).toBe("1.2 G")
        expect(strBigInt).toBe("1234567890 Y")
    })
})
