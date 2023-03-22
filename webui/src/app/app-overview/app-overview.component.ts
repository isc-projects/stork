import { Component, Input } from '@angular/core'
import { App, AppAccessPoint, ServicesService } from '../backend'
import { AuthService } from '../auth.service'

/**
 * A component that displays app overview.
 *
 * It comprises the information about the app and machine access points.
 */
@Component({
    selector: 'app-app-overview',
    templateUrl: './app-overview.component.html',
    styleUrls: ['./app-overview.component.sass'],
})
export class AppOverviewComponent {
    /**
     * Pointer to the structure holding the app information.
     */
    @Input() app: App = null

    /**
     * The authentication keys fetched from the API.
     */
    keys: Record<string, string> = {}

    constructor(private auth: AuthService, private servicesApi: ServicesService) {}

    /**
     * Conditionally formats an IP address for display.
     *
     * @param addr IPv4 or IPv6 address string.
     * @returns unchanged value if it is an IPv4 address or an IPv6 address
     *          surrounded by [ ].
     */
    formatAddress(addr: string): string {
        if (addr.length === 0 || !addr.includes(':') || (addr.startsWith('[') && addr.endsWith(']'))) {
            return addr
        }
        return `[${addr}]`
    }

    /**
     * Indicates if the authentication keys may be presented.
     * User must have privilege to show sensitive data and the application
     * must support authentication keys.
     */
    get canShowKeys() {
        return this.app.type === 'bind9' && this.auth.superAdmin()
    }

    /**
     * Indicates if the authentication key is loading.
     */
    isKeyLoading(accessPoint: AppAccessPoint) {
        return this.getKey(accessPoint) === null
    }

    /**
     * Indicates if the authentication key related to a given access point
     * is already fetched.
     */
    isKeyFetched(accessPoint: AppAccessPoint) {
        return this.getKey(accessPoint) != null
    }

    /**
     * Indicates if the authentication key related to a given access point
     * is already fetched and it is empty.
     */
    isKeyEmpty(accessPoint: AppAccessPoint) {
        return this.getKey(accessPoint) === ''
    }

    /**
     * Returns the authentication key related to a given access point.
     */
    getKey(accessPoint: AppAccessPoint) {
        return this.keys[accessPoint.type]
    }

    /**
     * Callback called when the user requests for the authentication key.
     */
    onAuthenticationKeyRequest(accessPoint: AppAccessPoint) {
        // Set loading state for a given access point.
        this.keys[accessPoint.type] = null
        this.servicesApi
            .getAccessPointKey(this.app.id, accessPoint.type)
            .toPromise()
            .then((key) => {
                // Set the new key.
                this.keys[accessPoint.type] = key
            })
            .catch((_) => {
                // Reset the loading state.
                this.keys[accessPoint.type] = undefined
            })
    }
}
