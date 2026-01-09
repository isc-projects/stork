/**
 * An interface to a representation of a daemon that can be selected
 * via a multi-select list.
 */
export interface SelectableDaemon {
    /**
     * Daemon ID.
     */
    id: number

    /**
     * Daemon name.
     */
    name: string

    /**
     * Daemon software version.
     */
    version: string

    /**
     * Daemon label presented in the multi-select list.
     */
    label: string
}
