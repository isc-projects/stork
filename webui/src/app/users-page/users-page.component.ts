import { Component, effect, OnDestroy, OnInit, signal, viewChild } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { ConfirmationService, MessageService, TableState, PrimeTemplate, MenuItem } from 'primeng/api'

import { AuthService, isInternalUser } from '../auth.service'
import { ServerDataService } from '../server-data.service'
import { UserSortField, UsersService } from '../backend'
import { debounceTime, firstValueFrom, lastValueFrom, Subject, Subscription } from 'rxjs'
import { getErrorMessage } from '../utils'
import { Group, User } from '../backend'
import { TabViewComponent } from '../tab-view/tab-view.component'
import { convertSortingFields, tableFiltersToQueryParams, tableHasFilter } from '../table'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { Table, TableModule } from 'primeng/table'
import { Router, RouterLink } from '@angular/router'
import { distinctUntilChanged, map } from 'rxjs/operators'
import { UserFormState } from '../forms/user-form'
import { ConfirmDialog } from 'primeng/confirmdialog'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { Button } from 'primeng/button'
import { ManagedAccessDirective } from '../managed-access.directive'
import { NgIf } from '@angular/common'
import { Tag } from 'primeng/tag'
import { IconField } from 'primeng/iconfield'
import { InputIcon } from 'primeng/inputicon'
import { InputText } from 'primeng/inputtext'
import { Checkbox } from 'primeng/checkbox'
import { UserFormComponent } from '../user-form/user-form.component'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { TableCaptionComponent } from '../table-caption/table-caption.component'
import { SplitButton } from 'primeng/splitbutton'

/**
 * Component for managing system users.
 */
@Component({
    selector: 'app-users-page',
    templateUrl: './users-page.component.html',
    styleUrls: ['./users-page.component.sass'],
    imports: [
        ConfirmDialog,
        BreadcrumbsComponent,
        TabViewComponent,
        Button,
        RouterLink,
        ManagedAccessDirective,
        TableModule,
        NgIf,
        Tag,
        PrimeTemplate,
        IconField,
        InputIcon,
        FormsModule,
        InputText,
        Checkbox,
        UserFormComponent,
        PlaceholderPipe,
        TableCaptionComponent,
        SplitButton,
    ],
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

    /**
     * Menu items of the splitButton which appears only for narrower viewports in the filtering toolbar.
     */
    toolbarButtons: MenuItem[] = []

    /**
     * This flag states whether user has privileges to create new user account.
     * This value comes from ManagedAccess directive which is called in the HTML template.
     */
    canCreateUser = signal<boolean>(false)

    /**
     * Effect signal reacting on user privileges changes and triggering update of the splitButton model
     * inside the filtering toolbar.
     */
    privilegesChangeEffect = effect(() => {
        if (this.canCreateUser()) {
            this._updateToolbarButtons()
        }
    })

    /**
     * Updates filtering toolbar splitButton menu items.
     * Based on user privileges some menu items may be disabled or not.
     * @private
     */
    private _updateToolbarButtons() {
        const buttons: MenuItem[] = [
            {
                label: 'Create User Account',
                icon: 'pi pi-plus',
                routerLink: '/users/new',
                disabled: !this.canCreateUser(),
            },
        ]
        this.toolbarButtons = [...buttons]
    }

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
                    map((f) => ({ ...f, value: f.value === '' ? null : f.value })), // replace empty string filter value with null
                    debounceTime(300),
                    distinctUntilChanged()
                )
                .subscribe((f) => {
                    // f.filterConstraint is passed as a reference to PrimeNG table filter FilterMetadata,
                    // so it's value must be set according to UI columnFilter value.
                    f.filterConstraint.value = f.value
                    this.router.navigate([], { queryParams: tableFiltersToQueryParams(this.table()) })
                })
        )

        firstValueFrom(this.serverData.getGroups()).then((groups) => (this.groups = groups.items ?? []))
        this._updateToolbarButtons()
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
        lastValueFrom(
            this.usersApi.getUsers(
                event.first,
                event.rows,
                event.filters['text'].value || null,
                ...convertSortingFields<UserSortField>(event)
            )
        )
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
            rejectButtonProps: { text: true, icon: 'pi pi-times' },
            acceptButtonProps: {
                icon: 'pi pi-check',
            },
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

    /**
     * Reference to the function so it can be used in html template.
     * @protected
     */
    protected readonly isInternalUser = isInternalUser

    /**
     * Reference to the function so it can be used in html template.
     * @protected
     */
    protected readonly tableHasFilter = tableHasFilter

    /**
     * Clears the PrimeNG table filtering. As a result, table pagination is also reset.
     * It doesn't reset the table sorting, if any was applied.
     */
    clearTableFiltering() {
        this.table()?.clearFilterValues()
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

    /**
     * Reference to an enum so it could be used in the HTML template.
     * @protected
     */
    protected readonly UserSortField = UserSortField
}
