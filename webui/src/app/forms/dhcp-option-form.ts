import { UntypedFormBuilder, UntypedFormGroup, Validators } from '@angular/forms'
import { IPType } from '../iptype'
import { gte } from 'semver'

/**
 * Creates a default form group for a DHCP option.
 *
 * When a new DHCP option form is added in any of the forms a
 * new form group for this option must be initialized. This
 * is a convenience function that creates such a form group
 * with required controls.
 *
 * @param keaVersionRange a tuple with the earliest and the latest Kea version
 * for the configured daemons.
 * @param universe IPv4 or IPv6 which is to determine the maximum
 * allowed option code value.
 * @returns created form group for an option.
 */
export function createDefaultDhcpOptionFormGroup(
    keaVersionRange: [string, string] | null,
    universe: IPType
): UntypedFormGroup {
    const fb = new UntypedFormBuilder()
    const group: Record<string, any> = {
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
        neverSend: [{ value: false, disabled: false }],
        optionFields: fb.array([]),
        suboptions: fb.array([]),
        unknown: fb.record<any>({}),
    }
    if (!keaVersionRange || gte(keaVersionRange[1], '2.7.4')) {
        group.clientClasses = [{ value: [], disabled: false }]
    }
    return fb.group(group)
}
