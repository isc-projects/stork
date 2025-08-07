import {
    AfterViewInit,
    Directive,
    ElementRef,
    EventEmitter,
    Input,
    Output,
    Renderer2,
    ViewContainerRef,
} from '@angular/core'
import { AccessType, AuthService, ManagedAccessEntity } from './auth.service'
import { Messages } from 'primeng/messages'

/**
 * This directive is meant to check authorization privileges for given entity.
 * In case of no privileges it will alter the component rendering depending on detected component type.
 * For most of the cases it will disable the UI element and the element will no longer be clickable.
 */
@Directive({
    selector: '[appAccessEntity]',
    standalone: true,
})
export class ManagedAccessDirective implements AfterViewInit {
    /**
     * Identifies the entity for which the access will be checked.
     */
    @Input({ required: true }) appAccessEntity: ManagedAccessEntity

    /**
     * Required access type to access the entity. Possible types follow CRUD naming convention:
     * create, read, update, delete.
     * Defaults to 'read' access type.
     */
    @Input() appAccessType: AccessType = 'read'

    /**
     * Optional input boolean flag which simplifies the directive usage. Defaults to false.
     * When set to true, it means that the component will not be displayed at all in case of lack of privileges.
     * When set to false (default), it means that the component will be rendered as disabled (if the component is a PrimeNG element),
     * or warning message will be displayed informing of lack of privileges.
     */
    @Input() appHideIfNoAccess: boolean = false

    /**
     * Output boolean property emitting whenever hasAccess changes.
     */
    @Output() appHasAccess: EventEmitter<boolean> = new EventEmitter()

    /**
     * Title content set for disabled buttons.
     * @private
     */
    private readonly _title = 'This component is disabled due to lack of privileges'

    constructor(
        private authService: AuthService,
        private elementRef: ElementRef,
        private renderer: Renderer2,
        private viewRef: ViewContainerRef
    ) {}

    ngAfterViewInit(): void {
        const hasAccess = this.authService.hasPrivilege(this.appAccessEntity, this.appAccessType)
        this.appHasAccess.emit(hasAccess)
        if (!hasAccess) {
            if (this.appHideIfNoAccess) {
                // Replace the element with an empty inline span.
                this.htmlElement.innerText = ''
                this.htmlElement.outerHTML = '<span></span>'
                return
            }

            // If this is a PrimeNG element...
            if (this.htmlElement.classList.contains('p-element')) {
                this.setDisabledClasses(this.htmlElement)

                // If this is a <button> element with PrimeNG pButton directive applied...
                if (this.htmlElement.nodeName.toUpperCase() === 'BUTTON') {
                    this.setDisabledAttributes(this.htmlElement)
                    return
                }

                // This is other PrimeNG element, e.g. <p-button>, <p-tristatecheckbox>, <p-inputswitch>, etc.
                this.htmlElement.querySelectorAll('.p-component').forEach((el) => {
                    // For all elements inside with .p-component class add disabled classes.
                    this.setDisabledClasses(<HTMLElement>el)
                })
                this.htmlElement.querySelectorAll('input,button').forEach((el) => {
                    // Set attributes and classes for all input and button elements found inside.
                    this.setDisabledAttributes(<HTMLElement>el)
                    this.setDisabledClasses(<HTMLElement>el)
                })

                // p-tristatecheckbox has .p-checkbox inside which requires additional class.
                this.htmlElement.querySelector('.p-checkbox')?.classList.add('p-checkbox-disabled')
                return
            }

            const messages = this.viewRef.createComponent(Messages)
            messages.instance.severity = 'warn'
            messages.instance.closable = false
            messages.instance.value = [
                {
                    severity: 'warn',
                    summary: 'Access Denied',
                    detail: `You don\'t have ${this.appAccessType} privileges to display this UI component.`,
                    closable: false,
                },
            ]
            this.htmlElement.replaceChildren(messages.instance.el.nativeElement)
        }
    }

    /**
     * Convenience getter returning native element of the element for which this directive is used.
     * @private
     */
    private get htmlElement(): HTMLElement {
        return this.elementRef.nativeElement as HTMLElement
    }

    /**
     * Sets attributes of the button to make it disabled and to show short feedback while hovering mouse over the button.
     * @param el button HTML element
     * @private
     */
    private setDisabledAttributes(el: HTMLElement): void {
        this.renderer.setAttribute(el, 'disabled', 'disabled')
        this.renderer.setAttribute(el, 'title', this._title)
    }

    /**
     * Adds classes to the HTML element to style it as disabled.
     * @param el HTML element where classes will be added
     * @private
     */
    private setDisabledClasses(el: HTMLElement): void {
        this.renderer.addClass(el, 'p-disabled')
        this.renderer.addClass(el, 'app-disabled')
    }
}
