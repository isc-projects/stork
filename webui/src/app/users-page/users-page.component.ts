import { Component, OnDestroy, OnInit, viewChild } from '@angular/core'
import { UntypedFormGroup } from '@angular/forms'
import { ConfirmationService, MessageService, TableState } from 'primeng/api'

import { AuthService } from '../auth.service'
import { ServerDataService } from '../server-data.service'
import { UsersService } from '../backend'
import { debounceTime, firstValueFrom, lastValueFrom, Subject, Subscription } from 'rxjs'
import { getErrorMessage } from '../utils'
import { Group, User } from '../backend'
import { TabViewComponent } from '../tab-view/tab-view.component'
import { tableFiltersToQueryParams, tableHasFilter } from '../table'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { Table } from 'primeng/table'
import { Router } from '@angular/router'
import { distinctUntilChanged, map } from 'rxjs/operators'
import { UserFormState } from '../forms/user-form'

/**
 * Form validator verifying if the confirmed password matches the password
 * value.
 *
 * @param passwordKey Name of the key under which the password value can be
 *                    found in the form.
 * @param confirmPasswordKey Name of the key under which the confirmed
 *                           password can be found in the form.
 * @returns The validator function comparing the passwords.
 */
export function matchPasswords(passwordKey: string, confirmPasswordKey: string) {
    return (group: UntypedFormGroup): { [key: string]: any } => {
        const password = group.get(passwordKey)
        const confirmPassword = group.get(confirmPasswordKey)

        if (password?.value !== confirmPassword?.value) {
            return {
                mismatchedPasswords: true,
            }
        }

        return null
    }
}

/**
 * Form validator verifying if the confirmed password is different from the
 * previous password.
 *
 * @param oldPasswordKey Name of the key under which the old password value can
 *                       be found in the form.
 * @param newPasswordKey Name of the key under which the new password value can
 *                       be found in the form.
 * @returns The validator function comparing the passwords.
 */
export function differentPasswords(oldPasswordKey: string, newPasswordKey: string) {
    return (group: UntypedFormGroup): { [key: string]: any } => {
        const oldPassword = group.get(oldPasswordKey)
        const newPassword = group.get(newPasswordKey)

        if (oldPassword?.value === newPassword?.value) {
            return {
                samePasswords: true,
            }
        }

        return null
    }
}

/**
 * Indicates if the user in an active tab is managed by an internal
 * authentication service
 */
export function isInternalUser(user: User) {
    const authenticationMethodId = user.authenticationMethodId
    // Empty or null or internal.
    return !authenticationMethodId || authenticationMethodId === 'internal'
}

/**
 * Component for managing system users.
 */
@Component({
    selector: 'app-users-page',
    templateUrl: './users-page.component.html',
    styleUrls: ['./users-page.component.sass'],
})
export class UsersPageComponent implements OnInit, OnDestroy {
    breadcrumbs = [{ label: 'Configuration' }, { label: 'Users' }]

    groups: Group[] = []
    // users table
    users: User[] = []
    totalUsers: number = 0

    tabView = viewChild(TabViewComponent)

    table = viewChild(Table)

    userProvider: (id: number) => Promise<User> = (id) => lastValueFrom(this.usersApi.getUser(id))

    userFormProvider = () => new UserFormState()

    tabTitleProvider: (user: User) => string = (user: User) => user.login || user.email

    private _subscriptions: Subscription = new Subscription()

    constructor(
        private usersApi: UsersService,
        private msgSrv: MessageService,
        private serverData: ServerDataService,
        public auth: AuthService,
        private confirmService: ConfirmationService,
        private router: Router
    ) {}

    ngOnInit() {
        this._restoreTableRowsPerPage()

        this._subscriptions.add(
            this._tableFilter$
                .pipe(
                    map((f) => {
                        return { ...f, value: f.value ?? null }
                    }),
                    debounceTime(300),
                    distinctUntilChanged(),
                    map((f) => {
                        f.filterConstraint.value = f.value
                        // this.zone.run(() =>
                        this.router.navigate(
                            [],
                            { queryParams: tableFiltersToQueryParams(this.table()) }
                            // )
                        )
                    })
                )
                .subscribe()
        )

        firstValueFrom(this.serverData.getGroups()).then((groups) => (this.groups = groups.items ?? []))
    }

    ngOnDestroy() {
        this._tableFilter$.complete()
        this._subscriptions.unsubscribe()
    }

    /**
     * Loads system users from the database into the component.
     *
     * @param event Event object containing index of the first row, maximum number
     *              of rows to be returned and the filter text.
     */
    loadUsers(event) {
        lastValueFrom(this.usersApi.getUsers(event.first, event.rows, event.filters['text'].value || null))
            .then((data) => {
                this.users = data.items ?? []
                this.totalUsers = data.total ?? 0
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Loading user accounts failed',
                    detail: 'Failed to load user accounts from the database: ' + msg,
                    life: 10000,
                })
            })
    }

    /**
     * Displays a dialog to confirm user deletion.
     * @param id
     */
    confirmDeleteUser(id: number) {
        this.confirmService.confirm({
            message: 'Are you sure that you want to permanently delete this user?',
            header: 'Delete User',
            icon: 'pi pi-exclamation-triangle',
            rejectButtonProps: { text: true },
            accept: () => {
                this.deleteUser(id)
            },
        })
    }

    /**
     * Action invoked when existing user form is being deleted
     *
     * As a result of this action an existing user account is attempted to be
     * deleted.
     */
    deleteUser(id: number) {
        lastValueFrom(this.usersApi.deleteUser(id))
            .then((/* data */) => {
                this.msgSrv.add({
                    severity: 'success',
                    summary: 'User account deleted',
                    detail: 'Successfully deleted user account.',
                })
                this.tabView()?.onDeleteEntity(id)
            })
            .catch((err) => {
                const msg = getErrorMessage(err)
                this.msgSrv.add({
                    severity: 'error',
                    summary: 'Failed to delete user account',
                    detail: 'Failed to delete user account: ' + msg,
                    life: 10000,
                })
            })
    }

    /**
     * Return group name for the particular group id
     *
     * @param groupId group id for which the name should be returned.
     * @returns group name.
     */
    public getGroupName(groupId): string {
        // The super-admin group is well known and doesn't require
        // iterating over the list of groups fetched from the server.
        // Especially, if the server didn't respond properly for
        // some reason, we still want to be able to handle the
        // super-admin group.
        if (groupId === 1) {
            return 'super-admin'
        }
        for (const grp of this.groups) {
            if (grp.id === groupId) {
                return grp.name
            }
        }
        return 'unknown'
    }

    protected readonly isInternalUser = isInternalUser
    protected readonly tableHasFilter = tableHasFilter

    /**
     * Clears the PrimeNG table state (filtering, pagination are reset).
     */
    clearTableState() {
        this.table()?.clear()
        this.router.navigate([])
    }

    /**
     * RxJS Subject used for filtering table data based on UI filtering form inputs (text inputs, checkboxes, dropdowns etc.).
     * @private
     */
    private _tableFilter$ = new Subject<{ value: any; filterConstraint: FilterMetadata }>()

    /**
     *
     * @param value
     * @param filterConstraint
     * @param debounceMode
     */
    filterTable(value: any, filterConstraint: FilterMetadata, debounceMode = true): void {
        if (debounceMode) {
            this._tableFilter$.next({ value, filterConstraint })
            return
        }

        filterConstraint.value = value
        this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.table()) })
    }

    /**
     * Clears single filter of the PrimeNG table.
     * @param filterConstraint filter metadata to be cleared
     */
    clearFilter(filterConstraint: any) {
        filterConstraint.value = null
        this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.table()) })
    }

    /**
     * Keeps number of rows per page in the table.
     */
    rows: number = 10

    /**
     * Key to be used in browser storage for keeping table state.
     * @private
     */
    private readonly _tableStateStorageKey = 'users-table-state'

    /**
     * Stores only rows per page count for the table in user browser storage.
     */
    storeTableRowsPerPage(rows: number) {
        const state: TableState = { rows: rows }
        const storage = this.table()?.getStorage()
        storage?.setItem(this._tableStateStorageKey, JSON.stringify(state))
    }

    /**
     * Restores only rows per page count for the table from the state stored in user browser storage.
     * @private
     */
    private _restoreTableRowsPerPage() {
        const stateString = localStorage.getItem(this._tableStateStorageKey)
        if (stateString) {
            const state: TableState = JSON.parse(stateString)
            this.rows = state.rows ?? 10
        }
    }
}
