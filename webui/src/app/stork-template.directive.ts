import { Directive, Input, TemplateRef } from '@angular/core'

@Directive({
    selector: '[appTemplate]',
    standalone: true,
})
export class StorkTemplateDirective {
    /**
     * Name of the Stork template.
     */
    @Input('appTemplate') templateName: string

    /**
     * Directive class constructor.
     * @param template embedded Stork template content
     */
    constructor(public template: TemplateRef<any>) {}

    /**
     * Returns the name of the Stork template.
     */
    getName(): string {
        return this.templateName
    }
}
