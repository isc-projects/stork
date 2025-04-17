import { AfterContentInit, Component, ContentChildren, Input, OnInit, QueryList, TemplateRef } from '@angular/core'
import { StorkTemplateDirective } from '../stork-template.directive'
import { AuthService } from '../auth.service'

@Component({
    selector: 'app-managed-access',
    templateUrl: './managed-access.component.html',
    styleUrl: './managed-access.component.sass',
})
export class ManagedAccessComponent implements AfterContentInit, OnInit {
    /**
     * Identifies the component for which the access will be checked.
     */
    @Input({ required: true }) key: string

    /**
     * List of Stork templates used for different rendering of the component based on received
     * privileges.
     */
    @ContentChildren(StorkTemplateDirective) templates: QueryList<StorkTemplateDirective>

    /**
     * Optional input boolean flag which simplifies the component usage.
     * When set to true, it means that the component will not be displayed at all in case of lack of write privileges,
     * instead of displaying alternative version with limited functionality.
     */
    @Input() hideOnNoWriteAccess: boolean = false

    writeAccessTemplate: TemplateRef<any>

    readAccessTemplate: TemplateRef<any>

    noAccessTemplate: TemplateRef<any>

    hasNoAccess: boolean = true
    hasReadAccess: boolean = false
    hasWriteAccess: boolean = false

    /**
     * Component class constructor.
     * @param authService service used to retrieve access privileges
     */
    constructor(private authService: AuthService) {}

    ngOnInit() {
        this.hasWriteAccess = this.authService.hasWritePrivilege(this.key)
        this.hasReadAccess = this.authService.hasReadPrivilege(this.key)
        // For now "no access" is disabled.
        this.hasNoAccess = false
    }

    ngAfterContentInit() {
        this.templates.forEach((item) => {
            switch (item.getName()) {
                case 'writeAccess':
                    this.writeAccessTemplate = item.template
                    break
                case 'readAccess':
                    this.readAccessTemplate = item.template
                    break
                case 'noAccess':
                default:
                    this.noAccessTemplate = item.template
                    break
            }
        })
    }
}
