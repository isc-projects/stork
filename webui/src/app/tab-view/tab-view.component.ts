import {
    booleanAttribute,
    Component,
    computed,
    contentChild,
    effect,
    Input,
    InputSignal,
    OnDestroy,
    OnInit,
    output,
    signal,
    TemplateRef,
} from '@angular/core'
import { Tab, TabList, TabPanel, TabPanels, Tabs } from 'primeng/tabs'
import { ActivatedRoute, EventType, ParamMap, Params, Router, RouterLink } from '@angular/router'
import { inject, input } from '@angular/core'
import { of, Subscription } from 'rxjs'
import { MessageService } from 'primeng/api'
import { TimesIcon } from 'primeng/icons'
import { NgClass, NgTemplateOutlet } from '@angular/common'
import { getErrorMessage } from '../utils'
import { parseBoolean, tableFiltersToQueryParams, tableHasFilter } from '../table'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { Table } from 'primeng/table'
import { filter, switchMap } from 'rxjs/operators'

/**
 * Enumeration of different Tab types.
 */
enum TabType {
    List = 1,
    New,
    Edit,
    Display,
}

/**
 * Data structure of the form that may be a part of the tab.
 */
type FormTab = {
    formState: FormState
    submitted: boolean
}

/**
 * Interface that must be implemented by a class defining form state (as of now there are two types of forms: create or edit entity).
 */
export interface FormState {
    transactionID: number
}

/**
 * Type defining data structure of a tab displayed and managed by the TabView component.
 */
export type ComponentTab = {
    title: string
    value: number
    route?: string | undefined
    icon?: string | undefined
    entity: { [key: string]: any }
    tabType: TabType
    form?: FormTab | undefined
}

/**
 * ID of the tab where create new entity form exists.
 */
const NEW_ENTITY_FORM_TAB_ID = -1

/**
 * Sanitizes given route path. Makes sure that the path has a trailing slash.
 * @param value path to be sanitized
 */
function sanitizePath(value: string | undefined): string | undefined {
    if (!value) {
        return undefined
    }

    if (value?.endsWith('/')) {
        return value
    } else {
        return `${value}/`
    }
}

/**
 * Component responsible for displaying clickable tabs and tabs' content below in separate panels.
 * It implements the logic for opening, closing, reopening already existing tabs etc.
 * It implements the logic of the forms that may be part of the tabs.
 */
@Component({
    selector: 'app-tab-view',
    standalone: true,
    imports: [Tabs, TabList, Tab, TabPanels, TabPanel, RouterLink, TimesIcon, NgTemplateOutlet, NgClass],
    templateUrl: './tab-view.component.html',
    styleUrl: './tab-view.component.sass',
})
export class TabViewComponent<TEntity, TForm extends FormState> implements OnInit, OnDestroy {
    /**
     * Input flag which determines whether the tabs are closable (close button is displayed next to the tab title).
     * Defaults to true.
     */
    closableTabs = input(true, { transform: booleanAttribute })

    /**
     * Input flag which determines whether for all entities that exist in the entitiesCollection array, a tab should
     * be created and open when the component is initialized.
     * Defaults to false.
     */
    initWithEntitiesInCollection = input(false, { transform: booleanAttribute })

    /**
     * Input flag which determines whether when opening new tab for an entity, table entitiesCollection
     * should be searched first or not for existing entity.
     * When set to true, every time entityProvider will be called (if available) to retrieve the entity.
     * Otherwise, this step may be skipped, if the entity exists in the entitiesCollection of the table.
     * Defaults to false.
     */
    forceAsyncEntityFetch = input(false, { transform: booleanAttribute })

    /**
     * Input string which holds the title displayed on the first tab.
     * Defaults to 'All'.
     */
    firstTabTitle = input('All')

    /**
     * Input string which holds the route end to be appended to the router link path of the first tab.
     * Defaults to 'all'.
     */
    firstTabRouteEnd = input('all')

    /**
     * Input string which holds the route end to be appended to the router link path of the tab with adding a new entity form.
     * Defaults to 'new'.
     */
    newEntityTabRouteEnd = input('new')

    /**
     * Input string which holds the route fragment to be appended to the router link path of the first tab.
     * When detected during navigation, if there exists a table of entities on the first tab, table's data will not be reloaded.
     * This is to prevent backend load when switching between tabs.
     * Defaults to 'tab-navigation'.
     */
    tabNavigationRouteFragment = input('tab-navigation')

    /**
     * Input string which holds the route fragment to be appended to the router link path of the tab with editing entity form.
     * Defaults to 'edit'.
     */
    editEntityRouteFragment = input('edit')

    /**
     * Input string which holds the router link base path for all the tabs.
     * When provided, the tab view is using clickable tabs
     * as clickable router links (router navigation will happen after clicking the tab).
     * Defaults to undefined value, which means that router links are not used by default.
     */
    routePath = input(undefined, { transform: sanitizePath })

    /**
     * Route path used for the router link of the first tab. It is computed using routePath and firstTabRouteEnd inputs.
     * It remains undefined if the routePath input is undefined.
     */
    firstTabRoute = computed(() => (this.routePath() ? this.routePath() + this.firstTabRouteEnd() : undefined))

    /**
     * Route path used for the router link of the tab with adding a new entity form. It is computed using routePath and newEntityTabRouteEnd inputs.
     * It remains undefined if the routePath input is undefined.
     */
    newEntityTabRoute = computed(() => (this.routePath() ? this.routePath() + this.newEntityTabRouteEnd() : undefined))

    /**
     * Query parameters used for the router link of the first tab.
     */
    firstTabQueryParams = signal<Params>(undefined)

    /**
     * Collection of entities for which the tabs are created and displayed.
     * If #table PrimeNG table was found as content child, this collection refers to the table entities; otherwise
     * it refers to the entities input.
     */
    entitiesCollection = computed<TEntity[]>(() => this.entitiesTable()?.value || this.entities())

    /**
     * Input array of entities.
     */
    entities = input<TEntity[]>(undefined)

    /**
     * Field name used to extract entity identifier value.
     * The identifier value is also used as a tab identifier value.
     * Defaults to 'id'.
     */
    entityIDKey = input('id')

    /**
     * Field name used to extract entity title value.
     * The title value is displayed as the tab title.
     * Defaults to 'name'.
     */
    entityTitleKey = input('name')

    /**
     * Field name used to extract entity icon.
     * The icon value is used to display an icon on the tab.
     * If the field does not exist in the entity or its value is undefined, the icon will not be displayed on the tab.
     * Defaults to 'icon'.
     */
    entityTabIconKey = input('icon')

    /**
     * If provided, this input number is the entity ID for which the tab will be open and activated up front
     * when the component is initialized.
     * Defaults to undefined, which means that it is not used by default.
     */
    openEntityID = input<number>(undefined)

    /**
     * String input holding the name of the entity type.
     * E.g. Subnet, Shared network etc.
     * Defaults to 'Entity'.
     */
    entityTypeName = input('Entity')

    /**
     * Input table filters.
     */
    tableQueryParamFilters: InputSignal<{
        [k: string]: {
            type: 'numeric' | 'enum' | 'string' | 'boolean'
            matchMode: 'contains' | 'equals'
            enumValues?: string[]
            arrayType?: boolean
        }
    }> = input()

    /**
     * Defines the template for the first tab content (first tab is optional; very often it is a table with the entities).
     */
    firstTabTemplate = contentChild<TemplateRef<any> | undefined>('firstTab', { descendants: false })

    /**
     * Defines the template for the entity tab content.
     */
    entityTabTemplate = contentChild<TemplateRef<any> | undefined>('entityTab', { descendants: false })

    /**
     * Defines the template for the form tab content.
     */
    formTabTemplate = contentChild<TemplateRef<any> | undefined>('formTab')

    /**
     * PrimeNG table used as a table of entities, usually displayed in the first tab.
     * It is explicitly provided as component input.
     */
    inputTable = input<Table>()

    /**
     * PrimeNG table used as a table of entities, usually displayed in the first tab.
     * It is provided as a content child; it must have #table template reference to be found.
     */
    contentChildTable = contentChild<Table>('table')

    /**
     * PrimeNG table used as a table of entities, usually displayed in the first tab.
     * It refers to either contentChildTable or inputTable.
     */
    entitiesTable = computed(() => this.contentChildTable() || this.inputTable())

    /**
     * Input function used to asynchronously provide the entity based on given entity ID.
     * The function takes only one argument - entity ID and returns the Promise of the entity.
     */
    @Input() entityProvider: (id: number) => Promise<TEntity>

    /**
     * If the tab is supposed to provide the form functionality, this input function is used to provide new form instance.
     */
    @Input() newFormProvider: () => TForm

    /**
     * Input function used to provide custom title for the entities.
     * The function takes the entity as an argument.
     */
    @Input() entityTitleProvider: (entity: TEntity) => string = () => undefined

    /**
     * Input function used to call REST API endpoint responsible for deleting the transaction of the 'create new entity' form.
     * The function takes transactionID as an argument.
     */
    @Input() createTransactionDeleteAPICaller: (transactionID: number) => void

    /**
     * Input function used to call REST API endpoint responsible for deleting the transaction of the 'update existing entity' form.
     * The function takes entityID and transactionID as arguments.
     */
    @Input() updateTransactionDeleteAPICaller: (entityID: number, transactionID: number) => void

    /**
     * Output emitting the identifier value of currently active tab whenever the value changes.
     */
    activeTabChange = output<number>()

    /**
     * Output emitting whole data structure of the tab that was closed.
     */
    tabClosed = output<ComponentTab>()

    /**
     * Holds all open tabs.
     */
    openTabs: ComponentTab[] = []

    /**
     * Keeps the identifier value of currently active tab (this tab content is currently displayed,
     * while other open tabs' content is hidden).
     */
    activeTabEntityID: number = 0

    /**
     * Reference to the TabType enum so it could be used in the html template.
     * @protected
     */
    protected readonly TabType = TabType

    /**
     * Activated route injected to retrieve route params.
     * @private
     */
    private readonly activatedRoute = inject(ActivatedRoute)

    /**
     * Message service injected to display feedback messages in UI.
     * @private
     */
    private readonly messageService = inject(MessageService)

    /**
     * Router injected to trigger navigations.
     * @private
     */
    private readonly router = inject(Router)

    /**
     * RxJS subscription to be unsubscribed when the component is destroyed.
     * @private
     */
    private subscriptions: Subscription

    /**
     * Component lifecycle hook which inits the component.
     */
    ngOnInit(): void {
        console.log('storkTabViewComponent onInit', Date.now())

        if (this.initWithEntitiesInCollection()) {
            this.entitiesCollection().forEach((entity: TEntity) => {
                this.openTabs.push(this.createTab(entity))
            })
        }

        if (this.routePath()) {
            this.subscriptions = this.router.events
                .pipe(
                    filter((e, idx) => e.type === EventType.NavigationEnd || idx === 0),
                    switchMap((_e, idx) =>
                        of({
                            paramMap: this.activatedRoute.snapshot.paramMap,
                            queryParamMap: this.activatedRoute.snapshot.queryParamMap,
                            fragment: idx === 0 ? null : this.activatedRoute.snapshot.fragment,
                        })
                    )
                )
                .subscribe({
                    next: (snapshot) => {
                        const paramMap = snapshot.paramMap
                        const queryParamMap = snapshot.queryParamMap
                        const fragment = snapshot.fragment
                        console.log('router events emits next', paramMap, queryParamMap, fragment, Date.now())
                        const id = paramMap.get('id')
                        if (!id || id === this.firstTabRouteEnd()) {
                            console.log('no id in path or this is /all path - open first tab')
                            this.activeTabEntityID = 0
                            if (!this.entitiesTable()) {
                                console.log('no table yet')
                            }

                            if (
                                this.entitiesTable() &&
                                this.tableQueryParamFilters() &&
                                fragment !== this.tabNavigationRouteFragment()
                            ) {
                                const parsedFilters = this.queryParamMapToTableFilters(queryParamMap)
                                console.log(
                                    'qParams filter parsing',
                                    parsedFilters,
                                    'table has',
                                    this.entitiesTable()?.filters
                                )
                                this.filterTableUsingMultipleFilters(parsedFilters.filters)
                            }

                            return
                        }

                        if (id === this.newEntityTabRouteEnd()) {
                            this.openTab(NEW_ENTITY_FORM_TAB_ID)
                            this.loadTableDataIfEmpty()

                            return
                        }

                        const numericId = parseInt(id, 10)
                        if (!Number.isNaN(numericId)) {
                            this.openTab(numericId)
                            this.loadTableDataIfEmpty()

                            return
                        } else {
                            this.messageService.add({
                                detail: `Couldn't parse provided id ${id} to numeric value!`,
                                severity: 'error',
                                summary: `Error opening tab`,
                            })
                            this.goToFirstTab()
                            return
                        }
                    },
                })
            return
        }

        if (this.openEntityID()) {
            // openEntityID input was provided, so let's open this tab.
            this.openTab(this.openEntityID())
            return
        }

        this.goToFirstTab()
    }

    /**
     * Component lifecycle hook which destroys the component.
     */
    ngOnDestroy(): void {
        console.log('storkTabViewComponent onDestroy')
        this.subscriptions?.unsubscribe()
    }

    /**
     * Callback updating the tab title.
     * It may be called explicitly or when the updateTabTitleFn function from the entityTabTemplate context is called.
     * @param id tab ID which title should be updated
     * @param title updated title
     */
    onUpdateTabTitle = (id: number, title: string) => {
        console.log('onUpdateTabTitle', id, title)
        const existingTab = this.openTabs.find((tab) => tab.value === id)
        if (existingTab) {
            existingTab.title = title
        }
    }

    /**
     * Callback updating the tab entity.
     * It may be called explicitly or when the updateTabEntityFn function from the entityTabTemplate context is called.
     * To update the entity value, the callback is using the entity provided as a parameter (optional).
     * If the entity parameter was not provided, entityProvider is called to retrieve the updated entity.
     * @param id tab ID for which the entity should be updated
     * @param entity updated entity - optional; if provided, entityProvider will not be used
     */
    onUpdateTabEntity = (id: number, entity?: TEntity) => {
        if (!this.entityProvider && !entity) {
            return
        }

        const existingTab = this.openTabs.find((tab) => tab.value === id)
        const existingEntityInCollectionIdx = this.entitiesCollection()?.findIndex(
            (entity) => this.getID(entity) === id
        )
        console.log('onUpdateTabEntity', id, existingTab, existingEntityInCollectionIdx)
        if (existingTab || existingEntityInCollectionIdx > -1) {
            if (entity) {
                this.updateTabEntity(existingTab, existingEntityInCollectionIdx, entity)
                return
            }

            this.entityProvider(id)
                .then((entity) => {
                    this.updateTabEntity(existingTab, existingEntityInCollectionIdx, entity)
                })
                .catch((error) => {
                    const msg = getErrorMessage(error)
                    this.messageService.add({
                        detail: `Error trying to update tab with id ${id} - ${msg}`,
                        severity: 'error',
                        summary: `Error updating tab`,
                    })
                })
        }
    }

    /**
     * Helper method performing repeatable code in onUpdateTabEntity.
     * @param tab tab where the entity is to be updated
     * @param entityInCollectionIndex index in the entitiesCollection of the entity to be updated
     * @param entity updated entity
     * @private
     */
    private updateTabEntity(tab: ComponentTab, entityInCollectionIndex: number, entity: TEntity) {
        if (tab) {
            tab.entity = entity
            tab.title = this.getTitle(entity)
        }

        if (entityInCollectionIndex > -1) {
            this.entitiesTable()
                ? this.entitiesTable().value.splice(entityInCollectionIndex, 1, entity)
                : this.entities().splice(entityInCollectionIndex, 1, entity)
        }
    }

    /**
     * Callback doing cleanup when the form component gets destroyed.
     * It may be called explicitly or when the destroyFormFn function from the formTabTemplate context is called.
     * If at this step destroyed form still has active create/update transaction, it calls the delete API endpoint.
     * It is useful when e.g. user has active tab with the form and navigates away to other view in Stork.
     * It requires provided inputs: createTransactionDeleteAPICaller and updateTransactionDeleteAPICaller.
     * @param formState state of the form being destroyed
     */
    onDestroyForm = (formState: TForm) => {
        const foundTab = this.openTabs.find((tab) => tab.form?.formState.transactionID === formState.transactionID)
        console.log('onDestroyForm - transaction ID', formState.transactionID, 'found tab', foundTab)
        if (foundTab) {
            foundTab.form.formState = formState // TODO: this probably doesn't make sense?
            if (foundTab.tabType === TabType.New && formState.transactionID) {
                this.createTransactionDeleteAPICaller(formState.transactionID)
            } else if (foundTab.tabType === TabType.Edit && formState.transactionID) {
                this.updateTransactionDeleteAPICaller(foundTab.value, formState.transactionID)
            }
        }
    }

    /**
     * Callback updating entity form state in open tabs when the form is submitted.
     * It may be called explicitly or when the submitFormFn function from the formTabTemplate context is called.
     * It marks the form as submitted to prevent the component from canceling
     * the transaction. Next, in case it was Edit form, the tab is changed to display the edited entity.
     * In case it was New entity form and submitted ID is known, the tab is changed to display the new entity.
     * In case it was New entity form and submitted ID is unknown, it closes the form tab.
     * @param formState form state that was submitted
     * @param entityID optional ID of the entity
     */
    onSubmitForm = (formState: TForm, entityID?: number) => {
        const foundTab = this.openTabs.find((tab) => tab.form?.formState.transactionID === formState.transactionID)
        console.log(
            'onSubmitForm - transactionID',
            formState.transactionID,
            'entityID',
            entityID,
            'found tab',
            foundTab
        )
        if (foundTab) {
            foundTab.form.submitted = true

            if (foundTab.tabType === TabType.Edit) {
                // Edit form was submitted, so let's stay on this tab, but let's change to Display type and refresh the entity.
                foundTab.tabType = TabType.Display
                foundTab.icon = undefined
                // TODO: which one?
                // foundTab.form.formState.transactionID = 0
                // foundTab.form = undefined
                foundTab.form = undefined
                this.onUpdateTabEntity(foundTab.value)
            } else if (foundTab.tabType === TabType.New) {
                this.forceLoadTableData()
                this.closeTab(NEW_ENTITY_FORM_TAB_ID)
                if (entityID) {
                    // New entity form submitted, and we have information about new entity ID. So let's display new entity.
                    this.openTab(entityID)
                }
            }
        }
    }

    /**
     * Callback updating entity tab in open tabs when the form is cancelled.
     * It may be called explicitly or when the cancelFormFn function from the formTabTemplate context is called.
     * If the create new entity form is cancelled, the tab is closed. If the entity edit form is cancelled,
     * the tab is turned into the display type tab.
     * If at this step cancelled form still has active create/update transaction, it calls the delete API endpoint.
     * It requires provided inputs: createTransactionDeleteAPICaller and updateTransactionDeleteAPICaller.
     * @param tabType tab type to determine form type
     * @param entityID optional ID of the entity;
     */
    onCancelForm = (tabType: TabType, entityID?: number) => {
        const foundTab = this.openTabs.find(
            (tab) => tab.value === (entityID ?? NEW_ENTITY_FORM_TAB_ID) && tab.tabType === tabType
        )
        console.log('onCancelForm', tabType, entityID, 'found tab', foundTab)
        if (!foundTab) {
            return
        }

        if (tabType === TabType.New) {
            console.log('onCancelForm - create form')
            if (foundTab.form?.formState.transactionID && !foundTab.form.submitted) {
                this.createTransactionDeleteAPICaller(foundTab.form.formState.transactionID)
                foundTab.form.formState.transactionID = 0
            }

            this.closeTab(NEW_ENTITY_FORM_TAB_ID)
        } else if (tabType === TabType.Edit) {
            console.log('onCancelForm - edit form')
            foundTab.tabType = TabType.Display
            foundTab.icon = undefined

            if (foundTab.form?.formState.transactionID && !foundTab.form.submitted) {
                this.updateTransactionDeleteAPICaller(entityID, foundTab.form.formState.transactionID)
                // TODO: shall it be done?
                // foundTab.form = undefined
                foundTab.form.formState.transactionID = 0
            }
        }
    }

    /**
     * Callback updating entity tab in open tabs when user begins to edit the entity i.e. the edit form is created.
     * It may be called explicitly or when the beginEntityEditFn function from the formTabTemplate context is called.
     * @param entityID ID of the entity to be edited
     */
    onBeginEntityEdit = (entityID: number) => {
        const foundTab = this.openTabs.find((tab) => tab.value === entityID)
        console.log('onBeginEntityEdit', entityID, 'found tab', foundTab)
        if (foundTab && foundTab.tabType !== TabType.Edit) {
            foundTab.tabType = TabType.Edit
            foundTab.icon = 'pi pi-pencil'
            if (!foundTab.form) {
                console.log('onBeginEntityEdit - no form yet, create new one')
                foundTab.form = { submitted: false, formState: this.newFormProvider() }
            } else {
                foundTab.form.submitted = false
                console.log('onBeginEntityEdit - there is a form that exists', foundTab.form)
            }
        }

        this.openTab(entityID)
    }

    /**
     * Deletes the entity from entities collection and closes this entity tab.
     * @param id ID of the entity to be deleted
     */
    onDeleteEntity = (id: number) => {
        const existingEntityInCollectionIdx = this.entitiesCollection()?.findIndex(
            (entity) => this.getID(entity) === id
        )
        if (existingEntityInCollectionIdx > -1) {
            this.entitiesTable()
                ? this.entitiesTable().value.splice(existingEntityInCollectionIdx, 1)
                : this.entities().splice(existingEntityInCollectionIdx, 1)
        }

        this.closeTab(id)
    }

    /**
     * Gets the entity from the tab with given ID.
     * @param id ID of the entity
     */
    getOpenTabEntity(id: number): TEntity {
        return this.openTabs.find((tab) => tab.value === id)?.entity as TEntity
    }

    /**
     * Opens (activates) the tab for given entity ID. If the tab was not open before, it is created and then activated.
     * @param entityID entity ID for which the tab is open
     */
    openTab(entityID: number) {
        // console.log('openTab', entityID)
        if (entityID === this.activeTabEntityID) {
            console.log('openTab', entityID, 'this tab is already active')
            return
        }
        const existingTabIndex = this.openTabs.findIndex((tab) => tab.value === entityID)
        if (existingTabIndex > -1) {
            console.log('openTab', entityID, 'this tab is already open, switch active tab')
            // this.router.navigate(['/communication', existingTabIndex])
            this.activeTabEntityID = entityID
            return
        }

        if (entityID === NEW_ENTITY_FORM_TAB_ID) {
            console.log('openTab - requested new entity form, it doesnt exist yet')
            this.openTabs = [...this.openTabs, this.createNewEntityFormTab()]
            this.activeTabEntityID = entityID
            return
        }

        console.log('openTab', entityID, 'need to fetch data and create new tab')
        let entityToOpen = undefined
        // First let's check entities collection. Maybe the entity is there.
        if (this.entitiesCollection() && !this.forceAsyncEntityFetch()) {
            console.log('openTab - collection check', this.entitiesCollection())
            entityToOpen = this.entitiesCollection().find((entity) => this.getID(entity) === entityID)
        }

        if (entityToOpen) {
            console.log('openTab - entity found in entitiesCollection')
        }

        // At this step the entity must be retrieved asynchronously.
        if (!entityToOpen && this.entityProvider) {
            console.log('openTab - retrieve entity using entityProvider')
            this.entityProvider(entityID)
                .then((entity) => {
                    this.openTabs = [...this.openTabs, this.createTab(entity)]
                    this.activeTabEntityID = entityID
                })
                .catch((error) => {
                    const msg = getErrorMessage(error)
                    this.messageService.add({
                        detail: `Error trying to open tab with id ${entityID} - ${msg}`,
                        severity: 'error',
                        summary: `Error opening tab`,
                    })
                    this.goToFirstTab()
                })
            return
            // console.log('result in parent from child callable', res)
        }

        if (!entityToOpen) {
            this.messageService.add({
                detail: `Couldn't find tab to open with id ${entityID}!`,
                severity: 'error',
                summary: `Error opening tab`,
            })
            this.goToFirstTab()
            return
        }

        this.openTabs = [...this.openTabs, this.createTab(entityToOpen)]
        this.activeTabEntityID = entityID
    }

    /**
     * Closes given tab.
     * @param entityID entity ID for which the tab is closed
     */
    closeTab(entityID: number) {
        console.log('closeTab', entityID)
        if (!this.closableTabs || entityID < NEW_ENTITY_FORM_TAB_ID) {
            return
        }

        const activeTabIndex = this.openTabs.findIndex((tab) => tab.value === this.activeTabEntityID)
        const tabToCloseIndex = this.openTabs.findIndex((tab) => tab.value === entityID)
        console.log(
            `tabToCloseIndex: ${tabToCloseIndex} activeTabIndex: ${activeTabIndex} activeTabEntityID: ${this.activeTabEntityID}`
        )
        if (tabToCloseIndex > -1) {
            const closedTab = this.openTabs.splice(tabToCloseIndex, 1)[0]
            if (
                closedTab.tabType === TabType.New &&
                this.createTransactionDeleteAPICaller &&
                closedTab.form?.formState.transactionID &&
                !closedTab.form.submitted
            ) {
                this.createTransactionDeleteAPICaller(closedTab.form.formState.transactionID)
            } else if (
                closedTab.tabType === TabType.Edit &&
                this.updateTransactionDeleteAPICaller &&
                closedTab.form?.formState.transactionID &&
                !closedTab.form.submitted
            ) {
                this.updateTransactionDeleteAPICaller(entityID, closedTab.form.formState.transactionID)
            }

            this.tabClosed.emit(closedTab)
            if (tabToCloseIndex <= activeTabIndex) {
                this.goToFirstTab(true)
            }
        }
    }

    /**
     * Activates first tab.
     * @param tabNavigation optional boolean flag; if set to true, when the router navigation to the first tab happens,
     * route fragment is set to mark the navigation as a tab-navigation (i.e. the table data in the first tab will not be reloaded).
     * Defaults to false.
     */
    goToFirstTab(tabNavigation = false) {
        if (this.routePath()) {
            console.log('go to first tab using router')
            this.router.navigate([this.firstTabRoute()], {
                queryParams: this.firstTabQueryParams(),
                fragment: tabNavigation ? this.tabNavigationRouteFragment() : undefined,
            })
        } else {
            console.log('go to first tab without using router')
            this.activeTabEntityID = this.openTabs[0]?.value || 0
        }
    }

    /**
     * This effect signal is used for dev purpose.
     */
    initializedTableEffect = effect(() => {
        if (!this.entitiesTable()) {
            return
        }
        console.log('content child #table', this.entitiesTable(), Date.now())
    })

    /**
     * Callback called whenever active tab changes.
     * @param activeTabID currently changed and active tab ID which is also the entity ID
     */
    onActiveTabChange(activeTabID: string | number) {
        console.log('storkTabViewComponent log onValueChange', activeTabID)
        this.activeTabChange.emit(activeTabID as number)
    }

    /**
     * Loads the entities table data only if it is empty.
     */
    loadTableDataIfEmpty() {
        if (
            this.entitiesTable() &&
            !tableHasFilter(this.entitiesTable()) &&
            !(this.entitiesTable().value ?? []).length
        ) {
            console.log('loadTableDataIfEmpty, init table because it seems to be empty', this.entitiesTable()?.value)
            const metadata = this.entitiesTable().createLazyLoadMetadata()
            this.entitiesTable().onLazyLoad.emit(metadata)
        }
    }

    /**
     * Force loads the entities table data.
     */
    forceLoadTableData() {
        if (this.entitiesTable()) {
            console.log('forceLoadTableData')
            const metadata = this.entitiesTable().createLazyLoadMetadata()
            this.entitiesTable().onLazyLoad.emit(metadata)
        }
    }

    /**
     * Creates a tab data structure based on given entity.
     * @param entity entity used to construct the tab
     * @private
     */
    private createTab(entity: TEntity): ComponentTab {
        return {
            title: this.getTitle(entity),
            value: this.getID(entity),
            entity: entity,
            route: this.routePath() ? this.routePath() + this.getID(entity) : undefined,
            icon: this.getIcon(entity),
            tabType: TabType.Display,
        }
    }

    /**
     * Creates a tab data structure for the tab with the form for adding a new entity.
     * @private
     */
    private createNewEntityFormTab(): ComponentTab {
        return {
            title: `New ${this.entityTypeName()}`,
            value: NEW_ENTITY_FORM_TAB_ID,
            entity: undefined,
            route: this.newEntityTabRoute(),
            icon: 'pi pi-pencil',
            tabType: TabType.New,
            form: { submitted: false, formState: this.newFormProvider() },
        }
    }

    /**
     * Gets the identifier value of the entity.
     * @param entity the entity used to retrieve the data
     * @private
     */
    private getID(entity: TEntity): number {
        return entity[this.entityIDKey()]
    }

    /**
     * Gets the title value of the entity.
     * @param entity the entity used to retrieve the data
     * @private
     */
    private getTitle(entity: TEntity): string {
        return this.entityTitleProvider(entity) || entity[this.entityTitleKey()]
    }

    /**
     * Gets the icon string value of the entity.
     * @param entity the entity used to retrieve the data
     * @return icon string or undefined if it was not found
     * @private
     */
    private getIcon(entity: TEntity): string | undefined {
        return entity[this.entityTabIconKey()] || undefined
    }

    /**
     * Converts queryParamMap to PrimeNG table filters.
     * @param queryParamMap queryParams map to be converted
     * @private
     */
    private queryParamMapToTableFilters(queryParamMap: ParamMap): {
        count: number
        filters: { [p: string]: FilterMetadata }
    } {
        let validFilters = 0
        let _queryParamFilters = {}
        for (const paramKey of queryParamMap.keys) {
            if (!(paramKey in this.tableQueryParamFilters())) {
                this.messageService.add({
                    severity: 'error',
                    summary: 'Wrong URL parameter value',
                    detail: `URL parameter ${paramKey} not supported!`,
                    life: 10000,
                })
                continue
            }

            const paramValues = this.tableQueryParamFilters()[paramKey].arrayType
                ? queryParamMap.getAll(paramKey)
                : [queryParamMap.get(paramKey)]
            for (const paramValue of paramValues) {
                if (!paramValue) {
                    continue
                }

                let parsedValue = null
                switch (this.tableQueryParamFilters()[paramKey].type) {
                    case 'numeric':
                        const numV = parseInt(paramValue, 10)
                        if (Number.isNaN(numV)) {
                            this.messageService.add({
                                severity: 'error',
                                summary: 'Wrong URL parameter value',
                                detail: `URL parameter ${paramKey} requires numeric value!`,
                                life: 10000,
                            })
                            break
                        }

                        parsedValue = numV
                        validFilters += 1
                        break
                    case 'boolean':
                        const booleanV = parseBoolean(paramValue)
                        if (booleanV === null) {
                            this.messageService.add({
                                severity: 'error',
                                summary: 'Wrong URL parameter value',
                                detail: `URL parameter ${paramKey} requires either true or false value!`,
                                life: 10000,
                            })
                            break
                        }

                        parsedValue = booleanV
                        validFilters += 1
                        break
                    case 'enum':
                        if (this.tableQueryParamFilters()[paramKey].enumValues?.includes(paramValue)) {
                            parsedValue = paramValue
                            validFilters += 1
                            break
                        }

                        this.messageService.add({
                            severity: 'error',
                            summary: 'Wrong URL parameter value',
                            detail: `URL parameter ${paramKey} requires one of the values: ${this.tableQueryParamFilters()[paramKey].enumValues.join(', ')}!`,
                            life: 10000,
                        })
                        break
                    case 'string':
                        parsedValue = paramValue
                        validFilters += 1
                        break
                    default:
                        this.messageService.add({
                            severity: 'error',
                            summary: 'Wrong URL parameter value',
                            detail: `URL parameter ${paramKey} of type ${this.tableQueryParamFilters()[paramKey].type} not supported!`,
                            life: 10000,
                        })
                        break
                }

                if (parsedValue !== null) {
                    const filterConstraint = {}
                    if (this.tableQueryParamFilters()[paramKey].arrayType) {
                        parsedValue = _queryParamFilters[paramKey]?.value
                            ? [..._queryParamFilters[paramKey]?.value, parsedValue]
                            : [parsedValue]
                    }

                    filterConstraint[paramKey] = {
                        value: parsedValue,
                        matchMode: this.tableQueryParamFilters()[paramKey].matchMode,
                    }
                    _queryParamFilters = { ..._queryParamFilters, ...filterConstraint }
                }
            }
        }

        return { count: validFilters, filters: _queryParamFilters }
    }

    /**
     * Filters PrimeNG table by applying all given filters at once.
     * @param filters PrimeNG table filters
     * @private
     */
    private filterTableUsingMultipleFilters(filters: { [x: string]: FilterMetadata | FilterMetadata[] }): void {
        this.entitiesTable()?.clearFilterValues()
        const metadata = this.entitiesTable()?.createLazyLoadMetadata()
        this.entitiesTable().filters = { ...metadata.filters, ...filters }
        const qp = tableFiltersToQueryParams(this.entitiesTable())
        console.log('apply queryParams to first tab', qp)
        this.firstTabQueryParams.set(qp)
        this.entitiesTable()?._filter()
    }
}
