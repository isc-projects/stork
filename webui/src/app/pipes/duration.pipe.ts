import { Pipe, PipeTransform } from '@angular/core'
import { durationToString } from '../utils'

@Pipe({ name: 'duration' })
export class DurationPipe implements PipeTransform {
    /**
     * Formats the duration into a human-readable string.
     * @param value Either a number (seconds) or a string in the format of "1h2m3.4567s"
     * @param short boolean flag indicating if the duration should be output
     *              using short (if true) or long format (if false).
     * @param fractionalDigits Maximum number of fractional digits to display.
     * @returns formatted string in the format of "1 hour 2 minutes 3.4 seconds"
     */
    transform(value: string | number, short = false, fractionalDigits = 1): string | number {
        return durationToString(value, short, fractionalDigits)
    }
}
