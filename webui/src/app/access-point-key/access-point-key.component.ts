import { Component, Input } from '@angular/core'
import { ServicesService } from '../backend'

/**
 * Control that allows to fetch the access point key on demand.
 */
@Component({
    selector: 'app-access-point-key',
    templateUrl: './access-point-key.component.html',
    styleUrls: ['./access-point-key.component.sass'],
})
export class AccessPointKeyComponent {
    constructor(private servicesApi: ServicesService) {}

    /**
     * Application ID.
     */
    @Input() appId: number

    /**
     * Access point type.
     */
    @Input() accessPointType: string

    /**
     * The authentication key fetched from the API.
     */
    key: string | null | undefined = undefined

    /**
     * Indicates if the authentication key is loading.
     */
    get isKeyLoading() {
        return this.key === null
    }

    /**
     * Indicates if the authentication key related to a given access point
     * is already fetched.
     */
    get isKeyFetched() {
        return this.key != null
    }

    /**
     * Indicates if the authentication key related to a given access point
     * is already fetched and it is empty.
     */
    get isKeyEmpty() {
        return this.key === ''
    }

    /**
     * Callback called when the user requests getting the authentication key.
     */
    onAuthenticationKeyRequest() {
        // Set loading state for a given access point.
        this.key = null
        this.servicesApi
            .getAccessPointKey(this.appId, this.accessPointType)
            .toPromise()
            .then((key) => {
                // Set the new key.
                this.key = key
            })
            .catch(() => {
                // Reset the loading state.
                this.key = undefined
            })
    }
}
