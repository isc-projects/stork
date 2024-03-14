import { Pipe, PipeTransform } from '@angular/core'

@Pipe({
    name: 'pluralize',
})
export class PluralizePipe implements PipeTransform {
    /**
     * Returns number and singular/plural form of the word.
     * It is possible to give custom plural form (e.g. mice/mouse).
     *
     * E.g.
     * nr=0
     * nr | pluralize:'host' => '0 hosts'
     *
     * nr=1
     * nr | pluralize:'host' => '1 host'
     *
     * nr=3
     * nr | pluralize:'host' => '3 hosts'
     *
     * nr=3
     * nr | pluralize:'mouse':'mice' => '3 mice'
     *
     * @param number number used to determine plural/singular form; should be nr>=0
     * @param singularText word for the singular form
     * @param pluralText word for the plural form; if omitted, it will be singularText with 's' suffix (host=>hosts)
     */
    transform(number: number, singularText: string, pluralText: string = null): string {
        const pluralWord = pluralText ? pluralText : `${singularText}s`
        return number === 0 || number > 1 ? `${number} ${pluralWord}` : `${number} ${singularText}`
    }
}
