import { FormBuilder, FormGroup, Validators } from '@angular/forms'

/**
 * Creates a default form group for a DHCP option.
 *
 * When a new DHCP option form is added in any of the forms a
 * new form group for this option must be initialized. This
 * is a convenience function that creates such a form group
 * with required controls.
 *
 * @returns created form group for an option.
 */
export function createDefaultDhcpOptionFormGroup(): FormGroup {
    const fb = new FormBuilder()
    return fb.group({
        optionCode: [{ value: null, disabled: false }, Validators.required],
        alwaysSend: [{ value: false }],
        optionFields: fb.array([]),
        suboptions: fb.array([]),
    })
}
