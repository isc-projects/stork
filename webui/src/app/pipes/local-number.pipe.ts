import { Pipe, PipeTransform } from '@angular/core'

/**
 * Custom implementation of the Angular's Decimal Pipe that supports BigInt.
 */
@Pipe({
    name: 'localNumber',
})
export class LocalNumberPipe implements PipeTransform {
    /**
     * Formats the number using a given locale. If the value is string,
     * it'll be converted to number. Returns null for null value or invalid
     * string.
     * @param value Number, numeric string or null
     * @param locale Target locale (optional).
     * @returns Formatted number.
     */
    transform(value: string | number | bigint | null, locale?: string): string | null {
        if (value == null) {
            return null
        }

        if (typeof value === 'string') {
            try {
                value = BigInt(value)
            } catch {
                return null
            }
        }

        return value.toLocaleString(locale)
    }
}
