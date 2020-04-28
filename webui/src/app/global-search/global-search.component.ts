import { Component, OnInit, ViewChild } from '@angular/core'

import { OverlayPanel } from 'primeng/overlaypanel'

import { SearchService } from '../backend/api/api'

/**
 * Component for handling global search. It provides box
 * for entering search text and a panel with results.
 */
@Component({
    selector: 'app-global-search',
    templateUrl: './global-search.component.html',
    styleUrls: ['./global-search.component.sass'],
})
export class GlobalSearchComponent implements OnInit {
    @ViewChild('searchResultsBox')
    searchResultsBox: OverlayPanel

    searchText: string
    searchResults: any

    constructor(protected searchApi: SearchService) {}

    ngOnInit(): void {
        this.searchResults = {
            subnets: { items: [] },
            sharedNetworks: { items: [] },
            hosts: { items: [] },
            machines: { items: [] },
            apps: { items: [] },
            users: { items: [] },
            groups: { items: [] },
        }
    }

    /**
     * Search for records server-side, in the database.
     */
    searchRecords(event) {
        const recordTypes = ['subnets', 'sharedNetworks', 'hosts', 'machines', 'apps', 'users', 'groups']

        if (this.searchText.length >= 2 || event.key === 'Enter') {
            this.searchApi.searchRecords(this.searchText).subscribe((data) => {
                for (const k of recordTypes) {
                    if (data[k] && data[k].items) {
                        this.searchResults[k] = data[k]
                    } else {
                        this.searchResults[k] = { items: [] }
                    }
                }
                this.searchResultsBox.show(event)
                // this is a workaround to fix position when content of overlay panel changes
                setTimeout(() => {
                    this.searchResultsBox.align()
                }, 100)
            })
        }
    }
}
