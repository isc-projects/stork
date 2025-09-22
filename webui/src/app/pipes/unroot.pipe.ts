import { Pipe, PipeTransform } from '@angular/core'
import { unrootZone } from '../utils'

@Pipe({
    name: 'unroot',
    standalone: false,
})
export class UnrootPipe implements PipeTransform {
    /**
     * Converts the root zone name or empty name to '(root)'.
     */
    transform(value: string | null | undefined): string {
        return unrootZone(value)
    }
}
