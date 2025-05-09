import { AfterViewInit, Directive, ElementRef, EventEmitter, Input, Output, Renderer2 } from '@angular/core'
import { AccessType, AuthService, PrivilegeKey } from './auth.service'

@Directive({
    selector: '[appManagedAccess]',
    standalone: true,
})
export class ManagedAccessDirective implements AfterViewInit {
    @Input('appManagedAccess') key: PrivilegeKey
    @Input() accessType: AccessType = 'read'
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
        const hasAccess = this.authService.hasPrivilege(this.key, this.accessType)
        this.hasAccess.emit(hasAccess)
        if (!hasAccess) {
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
