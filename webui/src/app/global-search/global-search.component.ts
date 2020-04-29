import { Component, OnInit, ViewChild } from '@angular/core'

import { OverlayPanel } from 'primeng/overlaypanel'

import { SearchService } from '../backend/api/api'

const recordTypes = ['subnets', 'sharedNetworks', 'hosts', 'machines', 'apps', 'users', 'groups']

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
        this.resetResults()
    }

    /**
     * Reset results to be empty.
     */
    resetResults() {
        this.searchResults = {}
        for (const rt of recordTypes) {
            this.searchResults[rt] = { items: [] }
        }
    }

    /**
     * Search for records server-side, in the database.
     */
    searchRecords(event) {
        if (event.key === 'Escape') {
            this.resetResults()
            this.searchText = ''
            this.searchResultsBox.hide()
        } else if (this.searchText.length >= 2 || event.key === 'Enter') {
            this.searchApi.searchRecords(this.searchText).subscribe((data) => {
                this.resetResults()
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
                }, 1000)
            })
        } else {
            this.resetResults()
        }
    }

    /**
     * Return true if there are no results.
     */
    noResults() {
        let count = 0
        for (const rt of recordTypes) {
            count += this.searchResults[rt].items.length
        }
        return count === 0
    }
}
