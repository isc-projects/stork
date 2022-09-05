import { UntypedFormBuilder, UntypedFormGroup, Validators } from '@angular/forms'
import { IPType } from '../iptype'

/**
 * Creates a default form group for a DHCP option.
 *
 * When a new DHCP option form is added in any of the forms a
 * new form group for this option must be initialized. This
 * is a convenience function that creates such a form group
 * with required controls.
 *
 * @param universe IPv4 or IPv6 which is to determine the maximum
 * allowed option code value.
 * @returns created form group for an option.
 */
export function createDefaultDhcpOptionFormGroup(universe: IPType): UntypedFormGroup {
    const fb = new UntypedFormBuilder()
    return fb.group({
        optionCode: [
            { value: null, disabled: false },
            [
                Validators.required,
                Validators.pattern('[0-9]*'),
                Validators.min(1),
                Validators.max(universe === IPType.IPv4 ? 255 : 65535),
            ],
        ],
        alwaysSend: [{ value: false, disabled: false }],
        optionFields: fb.array([]),
        suboptions: fb.array([]),
    })
}
