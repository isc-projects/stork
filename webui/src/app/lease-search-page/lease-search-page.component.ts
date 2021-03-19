import { Component, OnInit } from '@angular/core'
import { map } from 'rxjs/operators'

import { MessageService } from 'primeng/api'

import { DHCPService } from '../backend/api/api'
import { LocaltimePipe } from '../localtime.pipe'

/**
 * Component providing an input box to search for DHCP leases.
 *
 * User should type one of the lease properties in the input box in
 * the expected format. The component will send a request to the
 * Stork server to find a lease matching the specified text. The
 * server will attempt to find a matching lease on all monitored
 * Kea servers and may potentially return multiple lease instances.
 *
 * The returned leases are displayed in the table with expandable
 * rows containing the details of each lease. The component displays
 * a warning message when the server had issues with contacting any
 * of the servers. The matching leases found on all other servers
 * are displayed in the table below the warning message.
 *
 * Various help tips displayed by this component describe in detail
 * how to use the leases search mechanism.
 */
@Component({
    selector: 'app-lease-search-page',
    templateUrl: './lease-search-page.component.html',
    styleUrls: ['./lease-search-page.component.sass'],
})
export class LeaseSearchPageComponent implements OnInit {
    breadcrumbs = [{ label: 'DHCP' }, { label: 'Leases Search' }]

    /**
     * Boolean flag indicating if at least one search attempt has been made.
     *
     * It is used to control the hints displayed.
     */
    searched = false

    /**
     * Holds current text typed in the search box.
     */
    searchText = ''

    /**
     * Holds search text previously submitted.
     */
    lastSearchText = ''

    /**
     * Holds a list of leases found as a result of the previous search attempt.
     */
    leases: any[]

    /**
     * Holds a list of apps for which an error occurred during last search.
     *
     * If this list is non-empty a warning message is displayed listing
     * problematic apps.
     */
    erredApps: any[]

    /**
     * Component constructor.
     *
     * @param dhcpApi service used to contact the DHCP servers.
     */
    constructor(private msgService: MessageService, private dhcpApi: DHCPService) {}

    /**
     * Hook displayed during component initialization.
     *
     * It currently does nothing.
     */
    ngOnInit(): void {}

    /**
     * Attempts to search leases using the text specified in the input box.
     *
     * It sends a query to the Stork server to find leases by specified text.
     * This operation may take some time because Stork server has to contact
     * all monitored Kea servers, and sometimes send multiple commands to each
     * of them.
     *
     * If the operation is successful, the list of leases is displayed in the
     * results table. Otherwise, the leases list is cleared.
     *
     * A leading and trailing whitespace is ignored in the search text. If the
     * search text is empty the search is not performed.
     */
    searchLeases() {
        // Remove whitespace and ensure that a user entered any search text.
        const searchText = this.searchText.trim()
        if (searchText.length === 0) {
            return
        }
        // Remember the text used for search. It will be used to display
        // information accompanying the search results.
        this.lastSearchText = searchText
        this.dhcpApi
            .getLeases(searchText)
            .pipe(
                // Leases are fetched from Kea servers directly and they lack
                // unique identifiers. The unique identifiers are required as
                // the data keys in the expandable table.
                map((data) => {
                    if (data.items) {
                        // For each returned lease assign a unique id.
                        let id = 1
                        for (const lease of data.items) {
                            lease.id = id
                            id++
                        }
                    }
                    return data
                })
            )
            .subscribe(
                (data) => {
                    // Fetching leases successful.
                    this.leases = data.items
                    this.erredApps = data.erredApps
                    this.searched = true
                },
                (err) => {
                    // Fetching leases erred.
                    let msg = err.statusText
                    if (err.error && err.error.message) {
                        msg = err.error.message
                    }
                    this.msgService.add({
                        severity: 'error',
                        summary: 'Leases search erred',
                        detail: 'Leases search by ' + searchText + ' erred: ' + msg,
                        life: 10000,
                    })

                    this.leases = []
                    this.erredApps = []
                    this.searched = true
                }
            )
    }

    /**
     * Event handler triggered when a key is pressed with a search box.
     *
     * It starts leases search when Enter key is pressed.
     *
     * @param event event containing pressed key's name.
     */
    handleKeyUp(event) {
        switch (event.key) {
            case 'Enter': {
                this.searchLeases()
                break
            }
            default: {
                break
            }
        }
    }

    /**
     * Converts Kea lease state specified as a number to string.
     *
     * @param state lease state code.
     * @returns State name.
     */
    leaseStateAsText(state) {
        if (!state) {
            return 'Valid'
        }
        switch (state) {
            case 0: {
                return 'Valid'
            }
            case 1: {
                return 'Declined'
            }
            case 2: {
                return 'Expired/Reclaimed'
            }
            default: {
                break
            }
        }
        return 'Unknown'
    }
}
