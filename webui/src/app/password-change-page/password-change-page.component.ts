import { Component } from '@angular/core'

/**
 * This component allows the logged user to change the password.
 */
@Component({
    selector: 'app-settings-page',
    templateUrl: './password-change-page.component.html',
    styleUrls: ['./password-change-page.component.sass'],
})
export class PasswordChangePageComponent {
    breadcrumbs = [{ label: 'User Profile' }, { label: 'Password Change' }]
}
