import { Component, OnInit } from '@angular/core'
import { map } from 'rxjs/operators'
import { DHCPService } from '../backend/api/api'
import { LocaltimePipe } from '../localtime.pipe'

@Component({
    selector: 'app-lease-search-page',
    templateUrl: './lease-search-page.component.html',
    styleUrls: ['./lease-search-page.component.sass'],
})
export class LeaseSearchPageComponent implements OnInit {
    breadcrumbs = [{ label: 'DHCP' }, { label: 'Leases Search' }]

    searched = false
    searchText = ''
    lastSearchText = ''
    leases: any[]
    erredApps: any[]

    constructor(private dhcpApi: DHCPService) {}

    ngOnInit(): void {}

    searchLeases() {
        const searchText = this.searchText.trim()
        if (searchText.length === 0) {
            return
        }
        this.lastSearchText = searchText
        this.dhcpApi
            .getLeases(searchText)
            .pipe(
                map((data) => {
                    if (data.items) {
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
                    this.leases = data.items
                    this.erredApps = data.erredApps
                    this.searched = true
                },
                (error) => {
                    console.log(error)
                    this.leases = []
                    this.erredApps = []
                    this.searched = true
                }
            )
    }

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
