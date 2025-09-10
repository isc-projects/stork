import {
    booleanAttribute,
    Component,
    computed,
    contentChild,
    ContentChild,
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

enum TabType {
    List = 1,
    New,
    Edit,
    Display,
}

type FormTab = {
    formState: any
    submitted: boolean
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
 *
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
 */
@Component({
    selector: 'app-tab-view',
    standalone: true,
    imports: [Tabs, TabList, Tab, TabPanels, TabPanel, RouterLink, TimesIcon, NgTemplateOutlet, NgClass],
    templateUrl: './tab-view.component.html',
    styleUrl: './tab-view.component.sass',
})
export class TabViewComponent<TEntity, TForm> implements OnInit, OnDestroy {
    /**
     * Holds all open tabs.
     */
    openTabs: ComponentTab[] = []

    /**
     * Keeps the identifier value of currently active tab (this tab content is currently displayed,
     * while other open tabs' content is hidden).
     */
    activeTabEntityID: number = 0

    activeTabChange = output<number>()

    tabClosed = output<ComponentTab>()

    /**
     * Input flag which determines whether the tabs are closable (close button is displayed next to the tab title).
     * Defaults to true.
     */
    closableTabs = input(true, { transform: booleanAttribute })

    /**
     * Input flag which determines whether for all entities that exist in the entitiesCollection array a tab should
     * be created and open when the component is initialized.
     * Defaults to false.
     */
    initWithEntitiesInCollection = input(false, { transform: booleanAttribute })

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
     * If provided, this input number is the entity ID for which the tab will be open and activated when the component is initialized.
     * Defaults to undefined, which means that it is not used by default.
     */
    openEntityID = input<number>(undefined)

    /**
     * String input holding the name of the entity type.
     * E.g. Subnet, Shared network etc.
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
     * Input function used to asynchronously provide the entity based on given entity ID.
     * The function takes only one argument - entity ID and returns the Promise of the entity.
     */
    @Input() entityProvider: (id: number) => Promise<TEntity>

    @Input() newFormProvider: () => TForm

    @Input() entityTitleProvider: (entity: TEntity) => string = () => undefined

    /**
     * Defines the template for the first tab content (first tab is optional; very often it is a table with the entities).
     */
    @ContentChild('firstTab', { descendants: false }) firstTabTemplate: TemplateRef<any> | undefined

    /**
     * Defines the template for the entity tab content.
     */
    @ContentChild('entityTab', { descendants: false }) entityTabTemplate: TemplateRef<any> | undefined

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
     *
     * @param id
     */
    getOpenTabEntity(id: number) {
        return this.openTabs.find((tab) => tab.value === id)?.entity as TEntity
    }

    /**
     * Callback updating the tab title.
     * It is called when the updateTabTitleFn function from the entityTabTemplate context is called.
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
     * It is called when the updateTabEntityFn function from the entityTabTemplate or firstTabTemplate context is called.
     * The callback is using entityProvider to update the entity value or explicitly provided entity as a parameter.
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
                if (existingTab) {
                    existingTab.entity = entity
                }

                if (existingEntityInCollectionIdx > -1) {
                    this.entitiesTable()
                        ? this.entitiesTable().value.splice(existingEntityInCollectionIdx, 1, entity)
                        : this.entities().splice(existingEntityInCollectionIdx, 1, entity)
                }

                return
            }

            this.entityProvider(id)
                .then((entity) => {
                    if (existingTab) {
                        existingTab.entity = entity
                    }

                    if (existingEntityInCollectionIdx > -1) {
                        this.entitiesTable()
                            ? this.entitiesTable().value.splice(existingEntityInCollectionIdx, 1, entity)
                            : this.entities().splice(existingEntityInCollectionIdx, 1, entity)
                    }
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
     * Callback updating entity form state in open tabs when the form component gets destroyed.
     * It is called when the destroyFormFn function from the formTabTemplate context is called.
     * @param formState form state to be stored
     * @param tabType tab type to determine form type
     * @param entityID optional ID of the entity; it can only be provided for Edit type of form
     */
    onDestroyForm = (formState: TForm, tabType: TabType, entityID?: number) => {
        const extendEntityID = tabType === TabType.New ? NEW_ENTITY_FORM_TAB_ID : entityID
        const tabToUpdate = this.openTabs.find(
            (tab) => tab.form && tab.tabType === tabType && tab.value === extendEntityID
        )
        if (tabToUpdate) {
            console.log('onDestroyForm - form to update found', tabToUpdate, 'new one', formState)
            tabToUpdate.form.formState = formState
        }
    }

    /**
     * Callback updating entity form state in open tabs when the form is submitted.
     * It is called when the submitFormFn function from the formTabTemplate context is called.
     * It marks the form as submitted to prevent the component from canceling
     * the transaction. Next, it closes the form tab.
     * @param tabType tab type to determine form type
     * @param entityID optional ID of the entity; it can only be provided for Edit type of form
     */
    onSubmitForm = (tabType: TabType, entityID?: number) => {
        const extendEntityID = tabType === TabType.New ? NEW_ENTITY_FORM_TAB_ID : entityID
        const tabToUpdate = this.openTabs.find(
            (tab) => tab.form && tab.tabType === tabType && tab.value === extendEntityID
        )
        if (tabToUpdate) {
            console.log('onSubmitForm - form to update found', tabToUpdate)
            tabToUpdate.form.submitted = true
            this.closeTab(extendEntityID)
        }
    }

    /**
     * Callback updating entity tab in open tabs when the form is cancelled.
     * It is called when the cancelFormFn function from the formTabTemplate context is called.
     * If the event comes from the new entity form, the tab is closed. If the
     * event comes from the entity edit form, the tab is turned into the
     * display type tab.
     * @param tabType tab type to determine form type
     * @param entityID optional ID of the entity; it can only be provided for Edit type of form
     */
    onCancelForm = (tabType: TabType, entityID?: number) => {
        if (tabType === TabType.New) {
            console.log('onCancelForm - just close the form')
            this.closeTab(NEW_ENTITY_FORM_TAB_ID)
            return
        }

        const tabToUpdate = this.openTabs.find((tab) => tab.form && tab.value === entityID && tab.tabType === tabType)
        if (tabToUpdate) {
            console.log('onCancelForm - form tab to cancel found', tabToUpdate)
            tabToUpdate.tabType = TabType.Display
            tabToUpdate.icon = undefined
            tabToUpdate.form = undefined
        }
    }

    onBeginEntityEdit = (entityID: number) => {
        console.log('onBeginEntityEdit', entityID)
        const existingTab = this.openTabs.find((tab) => tab.value === entityID)
        console.log('onBeginEntityEdit found tab', existingTab)
        if (existingTab && existingTab.tabType !== TabType.Edit) {
            existingTab.tabType = TabType.Edit
            existingTab.icon = 'pi pi-pencil'
            if (!existingTab.form) {
                existingTab.form = { submitted: false, formState: this.newFormProvider() }
            }

            console.log('onBeginEntityEdit found tab after', existingTab)
        }

        this.openTab(entityID)
    }

    /**
     * Deletes the entity from entities collection and closes this entity tab.
     * @param id
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
        if (this.entitiesCollection()) {
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
            const closedTab = this.openTabs.splice(tabToCloseIndex, 1)
            this.tabClosed.emit(closedTab[0])
            if (tabToCloseIndex <= activeTabIndex) {
                this.goToFirstTab(true)
            }
        }
    }

    /**
     * Activates first tab.
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
     * Creates a tab data structure based on given entity.
     * @param entity entity used to construct the tab
     * @param tabType type of the tab; if not provided, it is by default set to type Display
     */
    createTab(entity: TEntity, tabType?: TabType): ComponentTab {
        return {
            title: this.getTitle(entity),
            value: this.getID(entity),
            entity: entity,
            route: this.routePath() ? this.routePath() + this.getID(entity) : undefined,
            icon: this.getIcon(entity),
            tabType: tabType ?? TabType.Display,
        }
    }

    /**
     * Creates a tab data structure for the tab with the form for adding a new entity.
     */
    createNewEntityFormTab(): ComponentTab {
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
     * Converts queryParamMap to PrimeNG table filters.
     * @param queryParamMap
     * @private
     */
    private queryParamMapToTableFilters(queryParamMap: ParamMap): {
        count: number
        filters: { [p: string]: FilterMetadata }
    } {
        // TODO: Move the queryParams filter validation logic to table.ts to replace existing, more complicated logic.
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
     * @param filters
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

    initializedTableEffect = effect(() => {
        if (!this.entitiesTable()) {
            return
        }
        console.log('content child #table', this.entitiesTable(), Date.now())
        // const queryParamFilters = this._parseQueryParams(this.route.snapshot.queryParamMap)
        // const metadata = this.contentChildTable().createLazyLoadMetadata()
        // const filters = { ...metadata.filters, ...queryParamFilters.filters }
        // this.contentChildTable().filters = filters
        // const newMeta = { ...metadata, ...filters }
        // this.contentChildTable().onLazyLoad.emit(newMeta)
    })

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
                            this.forceLoadTableData()

                            return
                        }

                        const numericId = parseInt(id, 10)
                        if (!Number.isNaN(numericId)) {
                            this.openTab(numericId)
                            this.forceLoadTableData()

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
     * Debug string logger.
     * @param event
     */
    logChange(event: string | number) {
        console.log('storkTabViewComponent log onValueChange', event)
        this.activeTabChange.emit(event as number)
    }

    forceLoadTableData() {
        if (
            this.entitiesTable() &&
            !tableHasFilter(this.entitiesTable()) &&
            !(this.entitiesTable().value ?? []).length
        ) {
            console.log('forceLoadTableData, init table because it seems to be empty', this.entitiesTable()?.value)
            const metadata = this.entitiesTable().createLazyLoadMetadata()
            this.entitiesTable().onLazyLoad.emit(metadata)
        }
    }

    protected readonly TabType = TabType
}
