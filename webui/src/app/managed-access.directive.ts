import { AfterViewInit, Directive, ElementRef, Input, Renderer2 } from '@angular/core'
import { AccessType, AuthService, PrivilegeKey } from './auth.service'

@Directive({
    selector: '[appManagedAccess]',
    standalone: true,
})
export class ManagedAccessDirective implements AfterViewInit {
    @Input('appManagedAccess') key: PrivilegeKey
    @Input() accessType: AccessType

    constructor(
        private authService: AuthService,
        private elementRef: ElementRef,
        private renderer: Renderer2
    ) {}

    ngAfterViewInit(): void {
        const hasAccess = this.authService.hasPrivilege(this.key, this.accessType)
        if (!hasAccess) {
            this.renderer.setProperty(this.elementRef.nativeElement, 'disabled', true)
            this.renderer.addClass(this.elementRef.nativeElement, 'p-disabled')
            this.renderer.setAttribute(this.elementRef.nativeElement, 'disabled', 'disabled')
            this.renderer.setAttribute(this.elementRef.nativeElement, 'aria-disabled', 'disabled')
        }
    }
}
