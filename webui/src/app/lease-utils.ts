/**
 * Convert a lease state enum value to the conventional string representation.
 * - 0 = Valid
 * - 1 = Declined
 * - 2 = Expired/Reclaimed
 * - 3 = Released
 * - 4 = Registered
 * @param state number An integer from 0-4 (inclusive) from Kea's lease state enum.
	 @return string The corresponding description of the lease state.
 */
export function stateToString(state: number): string {
    switch (state) {
        // For compatibility with the previous implementation.
        case null:
            return 'Valid'
        case 0:
            return 'Valid'
        case 1:
            return 'Declined'
        case 2:
            return 'Expired/Reclaimed'
        case 3:
            return 'Released'
        case 4:
            return 'Registered'
        default:
            return '(invalid state)'
    }
}
