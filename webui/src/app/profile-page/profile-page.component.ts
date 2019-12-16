import { Component, OnInit } from '@angular/core'
import { AuthService } from '../auth.service'

@Component({
    selector: 'app-profile-page',
    templateUrl: './profile-page.component.html',
    styleUrls: ['./profile-page.component.sass'],
})
export class ProfilePageComponent implements OnInit {
    currentUser = null

    constructor(private auth: AuthService) {
        this.auth.currentUser.subscribe(x => {
            this.currentUser = x
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

    /**
     * Returns group name the current user belongs to.
     */
    public get groupName(): string {
        if (this.currentUser.groups && this.currentUser.groups.length > 0) {
            return this.auth.groupName(this.currentUser.groups[0])
        }
        return '(not assigned to any groups)'
    }

    ngOnInit() {}
}
