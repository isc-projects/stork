import { Component, OnInit } from '@angular/core'
import { Router, ActivatedRoute } from '@angular/router'
import { map } from 'rxjs/operators'

import { MessageService } from 'primeng/api'

import { DHCPService } from '../backend/api/api'
import { getErrorMessage } from '../utils'

/**
 * Enumeration specifying the status of the leases search.
 *
 * The status indicates if the lease search was never started,
 * is in progress or was completed.
 */
enum LeasesSearchStatus {
    NotSearched,
    Searching,
    Searched,
}

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
    public Status = LeasesSearchStatus

    breadcrumbs = [{ label: 'DHCP' }, { label: 'Lease Search' }]

    /**
     * Leases search status indicator.
     *
     * Holds the information if any attempt to search a lease was
     * made, the search is in progress or the search was finished.
     * This information is used to produce the results table contents
     * when leases collection is empty.
     */
    searchStatus = this.Status.NotSearched

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
     * Flag indicating if the current search text is invalid.
     */
    invalidSearchText = false

    /**
     * Holds hint message displayed when search text is invalid.
     */
    invalidSearchTextError = ''

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
    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private msgService: MessageService,
        private dhcpApi: DHCPService
    ) {}

    /**
     * Hook displayed during component initialization.
     *
     * It uses a text query parameter value to perform leases search.
     * If the parameter is not present, the search is not triggered.
     */
    ngOnInit(): void {
        const searchText = this.route.snapshot.queryParamMap.get('text')
        if (searchText) {
            this.searchText = searchText.trim()
            this.validate()
            if (!this.invalidSearchText) {
                this.searchLeases()
            }
        }
        if (this.searchText.length === 0) {
            // Remove empty text parameter.
            this.router.navigate(['/dhcp/leases'])
        }
    }

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
        // Activate a spinner indicating that the search is in progress.
        this.searchStatus = this.Status.Searching
        this.leases = []
        this.erredApps = []
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
                    this.searchStatus = this.Status.Searched
                },
                (err) => {
                    // Fetching leases erred.
                    const msg = getErrorMessage(err)
                    this.msgService.add({
                        severity: 'error',
                        summary: 'Error searching leases',
                        detail: 'Error searching leases for ' + searchText + ' : ' + msg,
                        life: 10000,
                    })

                    this.leases = []
                    this.erredApps = []
                    this.searchStatus = this.Status.Searched
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
        // Remove the text parameter as soon as we start typing the query.
        if (this.route.snapshot.queryParamMap.has('text')) {
            this.router.navigate(['/dhcp/leases'])
        }
        this.validate()
        if (this.invalidSearchText) {
            return
        }
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
     * Decodes lease type.
     *
     * @param leaseType lease type returned by Kea.
     * @returns IPv6 address (IA_NA) or IPv6 prefix (IA_PD).
     */
    leaseTypeAsText(leaseType) {
        if (!leaseType) {
            return 'IPv4 address'
        }
        switch (leaseType) {
            case 'IA_NA':
                return 'IPv6 address (IA_NA)'
            case 'IA_PD':
                return 'IPv6 prefix (IA_PD)'
            default:
                break
        }
        return 'Unknown'
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

    /**
     * Checks if the current search text is valid.
     *
     * If the search text is invalid an appropriate error message is raised.
     * The following cases are considered invalid:
     * - partial IPv4 address,
     * - single colon,
     * - single trailing colon,
     * - single leading colon,
     * - whitespaces near colons,
     * - three or more consecutive colons,
     * - IPv6 address having more than one occurrence of double colons.
     */
    private validate() {
        const searchText = this.searchText.trim()
        if (searchText.length === 0) {
            this.clearSearchTextError()
            return
        }
        // Handle a special case when user is searching by lease state.
        let matches = searchText.match(/^state:\s*(\S*)\s*$/)
        if (matches && matches.length === 2) {
            const state = matches[1].trim()
            if (state.length === 0) {
                this.reportSearchTextError('Specify lease state.')
                return
            }
            if (state !== 'declined') {
                if ('declined'.indexOf(state) === 0) {
                    this.reportSearchTextError('Use state:declined to search declined leases.')
                } else {
                    this.reportSearchTextError('Searching leases in the ' + state + ' state is unsupported.')
                }
                return
            }
            this.clearSearchTextError()
            return
        }

        // Partial IPv4 address.
        let regexp =
            /^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){1,2}((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.{0,1}){0,1}$/
        if (regexp.test(searchText)) {
            this.reportSearchTextError('Please enter the complete IPv4 address.')
            return
        }
        // Single colon.
        if (searchText.length === 1 && searchText[0] === ':') {
            this.reportSearchTextError('Invalid single colon.')
            return
        }
        // Trailing colon.
        const lastTwo = searchText.slice(-2)
        if (lastTwo[1] === ':' && lastTwo[0] !== ':') {
            this.reportSearchTextError('Invalid trailing colon.')
            return
        }
        // Leading colon.
        if (searchText[0] === ':' && searchText[1] !== ':') {
            this.reportSearchTextError('Invalid leading colon.')
            return
        }
        // Whitespace near colon.
        regexp = /(?:\s+:|:\s+)/
        if (regexp.test(searchText)) {
            this.reportSearchTextError('Invalid whitespace near a colon.')
            return
        }
        // More than two consecutive colons.
        regexp = /:{3,}/
        if (regexp.test(searchText)) {
            this.reportSearchTextError('Invalid multiple consecutive colons.')
            return
        }
        // Invalid IPv6 address having two or more occurrences of ::.
        matches = searchText.match(/::/g)
        if (matches && matches.length > 1) {
            this.reportSearchTextError('Invalid IPv6 address.')
            return
        }
        // Everything is fine.
        this.clearSearchTextError()
    }

    /**
     * Indicates that search text is invalid.
     *
     * @param errorMessage error message displayed as a hint next to
     *                     next to the search box.
     */
    private reportSearchTextError(errorMessage) {
        this.invalidSearchText = true
        this.invalidSearchTextError = errorMessage
    }

    /**
     * Clears search text error message.
     */
    private clearSearchTextError() {
        this.invalidSearchText = false
        this.invalidSearchTextError = ''
    }
}
