import { Pipe, PipeTransform } from '@angular/core'

@Pipe({
    name: 'unroot',
})
export class UnrootPipe implements PipeTransform {
    /**
     * Converts the root zone name or empty name to '(root)'.
     */
    transform(value: string | null | undefined): string {
        if (value == null || value.trim() === '.') {
            return '(root)'
        }
        return value
    }
}
