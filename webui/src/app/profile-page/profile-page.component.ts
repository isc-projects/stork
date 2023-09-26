import { Component, OnDestroy, OnInit } from '@angular/core'
import { AuthService } from '../auth.service'
import { ServerDataService } from '../server-data.service'
import { User } from '../backend'
import { Subscription } from 'rxjs'

/**
 * This component is for displaying information about the user's account.
 */
@Component({
    selector: 'app-profile-page',
    templateUrl: './profile-page.component.html',
    styleUrls: ['./profile-page.component.sass'],
})
export class ProfilePageComponent implements OnInit, OnDestroy {
    breadcrumbs = [{ label: 'User Profile' }]

    currentUser: User = null
    private groups: any[]
    public groupName: string

    /**
     * List of subscriptions created by the component.
     */
    private subscriptions = new Subscription()

    constructor(
        private auth: AuthService,
        private serverData: ServerDataService
    ) {
        this.subscriptions.add(
            this.auth.currentUser.subscribe((user) => {
                this.currentUser = user
            })
        )
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
     * Unsubscribe the subscriptions.
     */
    ngOnDestroy(): void {
        this.subscriptions.unsubscribe()
    }

    /**
     * Returns header of the panel with user profile data.
     *
     * The header typically consists of the user first and last name.
     * If the first and last name appear to be the same, only one of
     * them is returned (typically the case for the admin account).
     */
    public get profilePanelHeader(): string {
        if (!!this.currentUser.name && !!this.currentUser.lastname) {
            return `${this.currentUser.name} ${this.currentUser.lastname}`
        } else if (!!this.currentUser.name) {
            return this.currentUser.name
        } else if (!!this.currentUser.lastname) {
            return this.currentUser.lastname
        } else if (!!this.currentUser.login) {
            return this.currentUser.login
        } else if (!!this.currentUser.email) {
            return this.currentUser.email
        } else {
            return this.currentUser.id.toString()
        }
    }
}
