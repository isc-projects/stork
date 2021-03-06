import { Component, OnInit } from '@angular/core'
import { AuthService } from '../auth.service'
import { ServerDataService } from '../server-data.service'

/**
 * This component is for displaying information about the user's account.
 */
@Component({
    selector: 'app-profile-page',
    templateUrl: './profile-page.component.html',
    styleUrls: ['./profile-page.component.sass'],
})
export class ProfilePageComponent implements OnInit {
    breadcrumbs = [{ label: 'User Profile' }]

    currentUser = null
    private groups: any[]
    public groupName: string

    constructor(private auth: AuthService, private serverData: ServerDataService) {
        this.auth.currentUser.subscribe((x) => {
            this.currentUser = x
        })
    }

    /**
     * Loads all groups from the server
     */
    ngOnInit() {
        this.serverData.getGroups().subscribe((data) => {
            if (data.items) {
                this.groups = data.items
                if (this.currentUser.groups && this.currentUser.groups.length > 0) {
                    // Set the group name to be displayed for the current user.
                    this.groupName = this.serverData.getGroupName(this.currentUser.groups[0], this.groups)
                }
            }
        })
    }

    /**
     * Returns header of the panel with user profile data.
     *
     * The header typically consists of the user first and last name.
     * If the first and last name appear to be the same, only one of
     * them is returned (typically the case for the admin account).
     */
    public get profilePanelHeader(): string {
        let hdr = this.currentUser.firstName
        if (this.currentUser.firstName !== this.currentUser.lastName) {
            hdr += ' ' + this.currentUser.lastName
        }
        return hdr
    }
}
