import { Directive, Input } from '@angular/core'

@Directive({
    selector: '[appTemplate]',
    standalone: true,
})
export class StorkTemplateDirective {
    /**
     * Name of the Stork template.
     */
    @Input('appTemplate') templateName: string
    constructor() {}

    /**
     * Returns the name of the Stork template.
     */
    getName(): string {
        return this.templateName
    }
}
