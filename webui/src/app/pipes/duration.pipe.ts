import { Pipe, PipeTransform } from '@angular/core'

@Pipe({
    name: 'duration',
    standalone: false,
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
        ms: 'millisecond',
        µs: 'microsecond',
        ns: 'nanosecond',
    }

    /**
     * Maximum number of fractional digits to display.
     */
    fractionalDigits = 1

    /**
     * Formats the duration into a human-readable string.
     * @param value Either a number (seconds) or a string in the format of "1h2m3.4567s"
     * @returns formatted string in the format of "1 hour 2 minutes 3.4 seconds"
     */
    transform(value: string | number): string | number {
        if (value == null) {
            return value
        }

        let durationStr: string
        // If value is a number, convert it to seconds
        if (typeof value === 'number') {
            const hours = Math.floor(value / 3600)
            const minutes = Math.floor((value % 3600) / 60)
            const seconds = value % 60
            const parts = []

            if (hours > 0) {
                parts.push(`${hours}h`)
            }
            if (minutes > 0) {
                parts.push(`${minutes}m`)
            }
            if (seconds > 0 || parts.length === 0) {
                parts.push(`${seconds}s`)
            }
            durationStr = parts.join('')
        } else {
            durationStr = value
        }

        const numbers: number[] = []
        const units: string[] = []

        const digits = []
        for (let i = 0; i < durationStr.length; i++) {
            const c = durationStr[i]
            if ((c >= '0' && c <= '9') || c === '.') {
                digits.push(c)
            } else {
                let unit = c
                const nextChar = durationStr[i + 1]
                if (nextChar === 's') {
                    unit += nextChar
                    i++
                }

                if (digits.length > 0) {
                    numbers.push(parseFloat(digits.join('')))
                    digits.length = 0
                }
                units.push(unit)
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
