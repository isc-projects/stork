import { Pipe, PipeTransform } from '@angular/core'

@Pipe({
    name: 'duration',
})
export class DurationPipe implements PipeTransform {
    /**
     * Map of Golang-style duration units to human-readable units.
     */
    units = {
        s: 'second',
        m: 'minute',
        h: 'hour',
        d: 'day',
    }

    /**
     * Maximum number of fractional digits to display.
     */
    fractionalDigits = 1

    /**
     * Formats the Golang-style duration into a human-readable string.
     * @param value String in the format of "1h2m3.4567s"
     * @returns formatted string in the format of "1 hour 2 minutes 3.4 seconds"
     */
    transform(value: string): string {
        if (value == null) {
            return value
        }

        const numbers: number[] = []
        const units: string[] = []

        const digits = []
        for (let c of value) {
            if ((c >= '0' && c <= '9') || c === '.') {
                digits.push(c)
            } else {
                if (digits.length > 0) {
                    numbers.push(parseFloat(digits.join('')))
                    digits.length = 0
                }
                units.push(c)
            }
        }

        const strings = []
        for (let i = 0; i < numbers.length; i++) {
            const number = numbers[i]
            const unit = units[i]

            if (number === 0) {
                continue
            }

            let unitString = this.units[unit] || unit
            if (number !== 1) {
                unitString += 's'
            }

            const numberString = !Number.isInteger(number) ? number.toFixed(this.fractionalDigits) : number.toString()
            strings.push(`${numberString} ${unitString}`)
        }

        if (strings.length === 0) {
            return '0 seconds'
        }

        return strings.join(' ')
    }
}
