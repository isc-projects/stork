import { AfterViewInit, Directive, ElementRef, EventEmitter, Input, Output, Renderer2 } from '@angular/core'
import { AccessType, AuthService, PrivilegeKey } from './auth.service'

@Directive({
    selector: '[appManagedAccess]',
    standalone: true,
})
export class ManagedAccessDirective implements AfterViewInit {
    /**
     * Identifies the entity for which the access will be checked.
     */
    @Input({ required: true }) appManagedAccess: PrivilegeKey

    /**
     * Required access type to access the entity. Possible types follow CRUD naming convention:
     * create, read, update, delete.
     * Defaults to 'read' access type.
     */
    @Input() accessType: AccessType = 'read'

    /**
     * Optional input boolean flag which simplifies the directive usage. Defaults to false.
     * When set to true, it means that the component will not be displayed at all in case of lack of privileges.
     * When set to false (default), it means that the component will be rendered as disabled (if the component is a PrimeNG element),
     * or warning message will be displayed informing of lack of privileges.
     */
    @Input() hideOnNoAccess: boolean = false

    /**
     * Output boolean property emitting whenever hasAccess changes.
     */
    @Output() hasAccess: EventEmitter<boolean> = new EventEmitter()

    constructor(
        private authService: AuthService,
        private elementRef: ElementRef,
        private renderer: Renderer2
    ) {}

    ngAfterViewInit(): void {
        const hasAccess = this.authService.hasPrivilege(this.appManagedAccess, this.accessType)
        this.hasAccess.emit(hasAccess)
        if (!hasAccess) {
            if (this.hideOnNoAccess) {
                this.elementRef.nativeElement.innerHTML = ''
                return
            }

            if (
                this.elementRef.nativeElement.classList.contains('p-element') ||
                this.elementRef.nativeElement.nodeName.toUpperCase() === 'I'
            ) {
                this.renderer.setProperty(this.elementRef.nativeElement, 'disabled', true)
                this.renderer.addClass(this.elementRef.nativeElement, 'p-disabled')
                this.renderer.setAttribute(this.elementRef.nativeElement, 'disabled', 'disabled')
                this.renderer.setAttribute(this.elementRef.nativeElement, 'aria-disabled', 'disabled')
                this.elementRef.nativeElement.querySelector('.p-checkbox')?.classList.add('p-checkbox-disabled')
                this.elementRef.nativeElement.querySelector('.p-inputswitch')?.classList.add('p-disabled')
                if (this.elementRef.nativeElement.querySelector('input')) {
                    this.renderer.setAttribute(
                        this.elementRef.nativeElement.querySelector('input'),
                        'disabled',
                        'disabled'
                    )
                }
            } else {
                this.elementRef.nativeElement.innerHTML =
                    '<div role="alert" class="p-messages p-component" aria-atomic="true" aria-live="assertive" data-pc-name="message">' +
                    '<div role="alert" class="p-message p-message-warn max-w-40rem">' +
                    '<div class="p-message-wrapper" data-pc-section="wrapper">' +
                    '<span class="p-message-summary" data-pc-section="summary" style="">You don\'t have ' +
                    this.accessType +
                    ' privileges to display this UI component.<br>If you think this is unexpected, please contact your Stork system administrator.</span>' +
                    '</div>' +
                    '</div>'
            }
        }
    }
}
