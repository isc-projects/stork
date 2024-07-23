import { Pipe, PipeTransform } from '@angular/core'
import { unhyphen } from '../utils'

@Pipe({
    name: 'unhyphen',
})
export class UnhyphenPipe implements PipeTransform {
    /**
     * Converts parameter names from JSON notation with hyphens to camel case.
     *
     * It removes hyphens and replaces them with spaces. All words following
     * the hyphens are converted to begin with a capital letter.
     *
     * @param value a name to be converted to camel case.
     * @returns converted name.
     */
    transform(value: string | null): string | null {
        if (value == null) {
            return value
        }
        return unhyphen(value)
    }
}
