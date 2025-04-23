import {
    AfterContentInit,
    Component,
    ContentChildren,
    EventEmitter,
    Input,
    OnInit,
    Output,
    QueryList,
    TemplateRef,
} from '@angular/core'
import { StorkTemplateDirective } from '../stork-template.directive'
import { AccessType, AuthService, PrivilegeKey } from '../auth.service'

@Component({
    selector: 'app-managed-access',
    templateUrl: './managed-access.component.html',
    styleUrl: './managed-access.component.sass',
})
export class ManagedAccessComponent implements AfterContentInit, OnInit {
    /**
     * Identifies the component for which the access will be checked.
     */
    @Input({ required: true }) key: PrivilegeKey

    /**
     * Required access type to display the component.
     * Defaults to 'write' access type.
     */
    @Input() accessType: AccessType = 'write'

    /**
     * List of Stork templates used for different rendering of the component based on received
     * privileges.
     */
    @ContentChildren(StorkTemplateDirective) templates: QueryList<StorkTemplateDirective>

    /**
     * Optional input boolean flag which simplifies the component usage. Defaults to true.
     * When set to false, it means that an optional version of the component shall be displayed with limited functionality,
     * due to no privileges of given type. The limited functionality component may be provided via ng-Template with appTemplate
     * directive value set to "noAccess". E.g. <ng-template appTemplate="noAccess">No privileges to display this component. Talk to your admin.</ng-template>.
     * In case it is not provided, default "ban" icon will be displayed with some tooltip explanation.
     *
     * When set to true, it means that the component will not be displayed at all in case of lack of privileges.
     */
    @Input() hideOnNoAccess: boolean = true

    /**
     * Template with content to be displayed when user has no required privileges.
     */
    noAccessTemplate: TemplateRef<any>

    /**
     * Template with content to be displayed when user has required privileges.
     */
    hasAccessTemplate: TemplateRef<any>

    /**
     * Boolean flag keeping state whether user has given type of privileges to access the component.
     */
    hasAccess: boolean = false

    /**
     * Output boolean property emitting whenever hasAccess changes.
     */
    @Output() hasAccessChanged: EventEmitter<boolean> = new EventEmitter()

    /**
     * Component class constructor.
     * @param authService service used to retrieve access privileges
     */
    constructor(private authService: AuthService) {}

    ngOnInit() {
        this.hasAccess = this.authService.hasPrivilege(this.key, this.accessType)

        this.hasAccessChanged.emit(this.hasAccess)
    }

    ngAfterContentInit() {
        this.templates.forEach((item) => {
            switch (item.getName()) {
                case 'noAccess':
                    this.noAccessTemplate = item.template
                    break
                case 'hasAccess':
                default:
                    this.hasAccessTemplate = item.template
                    break
            }
        })
    }
}
