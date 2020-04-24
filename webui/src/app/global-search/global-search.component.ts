import { Component, OnInit, ViewChild } from '@angular/core'

import { OverlayPanel } from 'primeng/overlaypanel'

import { SearchService } from '../backend/api/api'

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

    searchRecords(event) {
        if (this.searchText.length >= 2 && event.key === 'Enter') {
            this.searchApi.searchRecords(this.searchText).subscribe((data) => {
                for (const k of ['subnets', 'sharedNetworks', 'hosts', 'machines', 'apps', 'users', 'groups']) {
                    if (data[k] && data[k].items) {
                        this.searchResults[k] = data[k]
                    } else {
                        this.searchResults[k] = { items: [] }
                    }
                }
                this.searchResultsBox.hide()
                this.searchResultsBox.show(event)
            })
        }
    }
}
