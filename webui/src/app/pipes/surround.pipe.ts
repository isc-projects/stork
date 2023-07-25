import { Pipe, PipeTransform } from '@angular/core'

@Pipe({
    name: 'surround',
})
export class SurroundPipe implements PipeTransform {
    /**
     * Surround a given string with the left prefix and right suffix.
     * It has no effect if the value is null or undefined.
     */
    transform(value: string | null, left: string, right: string): string | null {
        if (value == null) {
            return value
        }
        return left + value + right
    }
}
