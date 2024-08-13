import { Pipe, PipeTransform } from '@angular/core'

@Pipe({
    name: 'pluck',
})
export class PluckPipe implements PipeTransform {
    /** Extracts a property value for a given key from each item of provided list.
     *
     * It improves the performance of rendering templates that iterate over a
     * list of objects because the Angular pipe system will only call the
     * transform method when the input value changes.
     */
    transform<K extends string, T extends Record<K, any>>(value: T[], key: K): T[K][] {
        if (value == null || value.map === undefined) {
            // It's not a list.
            return []
        }
        return value.map((v) => v[key])
    }
}
