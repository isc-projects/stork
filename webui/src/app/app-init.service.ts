import { Injectable } from '@angular/core'
import { UsersService } from './backend/api/users.service'

@Injectable({
    providedIn: 'root',
})
export class AppInitService {
    /**
     * Holds a list of system groups defined in the system.
     *
     * The groups are fetched from the database upon the UI reload.
     */
    public groups: any[]

    constructor(private usersService: UsersService) {}

    /**
     * Peforms application initialization.
     *
     * The actions performed by this functions are invoked prior to
     * initialization of any other components. This guarantees that
     * the components have access to the data initialized in this
     * function. In particular, the components which require access
     * to system groups can rely on those groups being initialized.
     */
    Init() {
        return new Promise<void>((resolve, reject) => {
            this.usersService.getGroups().subscribe(
                data => {
                    if (data.items) {
                        this.groups = data.items
                    }
                    resolve()
                },
                completed => {
                    resolve()
                }
            )
        })
    }
}
