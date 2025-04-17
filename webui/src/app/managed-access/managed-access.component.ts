import { Component, ContentChildren, Input, QueryList } from '@angular/core'
import { StorkTemplateDirective } from '../stork-template.directive'

@Component({
    selector: 'app-managed-access',
    templateUrl: './managed-access.component.html',
    styleUrl: './managed-access.component.sass',
})
export class ManagedAccessComponent {
    /**
     * Identifies the component for which the access will be checked.
     */
    @Input({ required: true }) key: string

    /**
     * List of Stork templates used for different rendering of the component based on received
     * privileges.
     */
    @ContentChildren(StorkTemplateDirective) templates: QueryList<StorkTemplateDirective>
}
