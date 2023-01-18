/**
 * An interface to a DHCP option definition.
 *
 * It uses the same format as Kea and Stork server
 * to represent an option definition.
 */
export interface DhcpOptionDef {
    code: number
    name: string
    space: string
    optionType: string
    array: boolean
    encapsulate: string
    recordTypes?: string[]
}
