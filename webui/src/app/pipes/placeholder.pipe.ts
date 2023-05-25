import { Pipe, PipeTransform } from '@angular/core'

@Pipe({
    name: 'placeholder',
})
export class PlaceholderPipe implements PipeTransform {
    // Returns a placeholder if the provided string is empty or unspecified
    // (null or undefined).
    transform(value: string, unspecified: string = '(not specified)', empty: string = '(empty)'): string {
        if (value == null) {
            return unspecified
        } else if (value == '') {
            return empty
        } else {
            return value
        }
    }
}
