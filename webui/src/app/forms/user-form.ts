import { FormState } from '../tab-view/tab-view.component'
import { UntypedFormGroup } from '@angular/forms'
import { User } from '../backend'

export class UserFormState implements FormState {
    /**
     * Not used in this form
     */
    transactionID: number = 0

    /**
     * A form group comprising all form controls, arrays and other form
     * groups (a parent group for the HostFormComponent form).
     */
    group: UntypedFormGroup

    /**
     * User that is being edited. For new user form it is undefined.
     */
    user: User
}
