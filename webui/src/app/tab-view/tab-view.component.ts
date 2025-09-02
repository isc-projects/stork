import {
    AfterViewInit,
    booleanAttribute,
    Component,
    computed,
    ContentChild,
    Input,
    InputSignal,
    model,
    OnDestroy,
    OnInit,
    TemplateRef,
} from '@angular/core'
import { Tab, TabList, TabPanel, TabPanels, Tabs } from 'primeng/tabs'
import { ActivatedRoute, ParamMap, Router, RouterLink } from '@angular/router'
import { inject, input } from '@angular/core'
import { combineLatest, debounceTime, Subscription } from 'rxjs'
import { MessageService } from 'primeng/api'
import { TimesIcon } from 'primeng/icons'
import { NgClass, NgTemplateOutlet } from '@angular/common'
import { getErrorMessage } from '../utils'
import { parseBoolean } from '../table'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { Table } from 'primeng/table'
import { filter } from 'rxjs/operators'

/**
 * Type defining data structure of a tab displayed and managed by the TabView component.
 */
type ComponentTab = {
    title: string
    value: number
    route?: string | undefined
    icon?: string
    entity: { [key: string]: any }
}

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
export class TabViewComponent<TEntity> implements OnInit, OnDestroy, AfterViewInit {
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
     * Collection of entities for which the tabs are created and displayed.
     */
    entitiesCollection = input<TEntity[]>(undefined)

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
     * Defaults to -1, which means that it is not used by default.
     */
    openEntityID = input(-1)

    supportedQueryParamFilters: InputSignal<{
        [k: string]: {
            type: 'numeric' | 'enum' | 'string' | 'boolean'
            matchMode: 'contains' | 'equals'
            enumValues?: string[]
            arrayType?: boolean
        }
    }> = input()

    /**
     * Boolean flag stating whether this component init is done or not.
     * @private
     */
    private _initDone: boolean = false

    /**
     * Input function used to asynchronously provide the entity based on given entity ID.
     * The function takes only one argument - entity ID and returns the Promise of the entity.
     */
    @Input() entityProvider: (id: number) => Promise<TEntity>

    /**
     * Defines the template for the first tab content (first tab is optional; very often it is a table with the entities).
     */
    @ContentChild('firstTab', { descendants: false }) firstTabTemplate: TemplateRef<any> | undefined

    /**
     * Defines the template for the entity tab content.
     */
    @ContentChild('entityTab', { descendants: false }) entityTabTemplate: TemplateRef<any> | undefined

    /**
     * Activated route injected to retrieve route params.
     * @private
     */
    private readonly route = inject(ActivatedRoute)

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
     * The callback is using entityProvider to update the entity value.
     * @param id tab ID for which the entity should be updated
     */
    onUpdateTabEntity = (id: number) => {
        if (!this.entityProvider) {
            return
        }

        const existingTab = this.openTabs.find((tab) => tab.value === id)
        const existingEntityInCollectionIdx = this.entitiesCollection()?.findIndex(
            (entity) => this.getID(entity) === id
        )
        console.log('onUpdateTabEntity', id, existingTab, existingEntityInCollectionIdx)
        if (existingTab || existingEntityInCollectionIdx > -1) {
            this.entityProvider(id)
                .then((entity) => {
                    if (existingTab) {
                        existingTab.entity = entity
                    }

                    if (existingEntityInCollectionIdx > -1) {
                        this.entitiesCollection().splice(existingEntityInCollectionIdx, 1, entity)
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

    loadTableDataOnInit = model(true)

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
        return entity[this.entityTitleKey()]
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

        console.log('openTab', entityID, 'need to fetch data and create new tab')
        let entityToOpen = undefined
        // First let's check entities collection. Maybe the entity is there.
        if (this.entitiesCollection()) {
            entityToOpen = this.entitiesCollection().find((entity) => this.getID(entity) === entityID)
        }

        // At this step the entity must be retrieved asynchronously.
        if (!entityToOpen && this.entityProvider) {
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
        if (!this.closableTabs || entityID <= 0) {
            return
        }

        const activeTabIndex = this.openTabs.findIndex((tab) => tab.value === this.activeTabEntityID)
        const tabToCloseIndex = this.openTabs.findIndex((tab) => tab.value === entityID)
        console.log(
            `tabToCloseIndex: ${tabToCloseIndex} activeTabIndex: ${activeTabIndex} activeTabEntityID: ${this.activeTabEntityID}`
        )
        if (tabToCloseIndex > -1) {
            this.openTabs.splice(tabToCloseIndex, 1)
            if (tabToCloseIndex <= activeTabIndex) {
                this.goToFirstTab()
            }
        }
    }

    /**
     * Activates first tab.
     */
    goToFirstTab() {
        if (this.routePath()) {
            console.log('go to first tab using router')
            this.router.navigate([this.firstTabRoute()])
        } else {
            console.log('go to first tab without using router')
            this.activeTabEntityID = this.openTabs[0]?.value || 0
        }
    }

    /**
     * Creates a tab data structure based on given entity.
     * @param entity entity used to construct the tab
     */
    createTab(entity: TEntity): ComponentTab {
        return {
            title: this.getTitle(entity),
            value: this.getID(entity),
            entity: entity,
            route: this.routePath() ? this.routePath() + this.getID(entity) : undefined,
            icon: this.getIcon(entity),
        }
    }

    // private _queryParamFilters: { [p: string]: FilterMetadata } = {}

    table: InputSignal<Table> = input()

    /**
     *
     * @param queryParamMap
     * @private
     */
    private _parseQueryParams(queryParamMap: ParamMap): { count: number; filters: { [p: string]: FilterMetadata } } {
        // TODO: Move the queryParams filter validation logic to table.ts to replace existing, more complicated logic.
        let validFilters = 0
        let _queryParamFilters = {}
        for (const paramKey of queryParamMap.keys) {
            if (!(paramKey in this.supportedQueryParamFilters())) {
                this.messageService.add({
                    severity: 'error',
                    summary: 'Wrong URL parameter value',
                    detail: `URL parameter ${paramKey} not supported!`,
                    life: 10000,
                })
                continue
            }

            const paramValues = this.supportedQueryParamFilters()[paramKey].arrayType
                ? queryParamMap.getAll(paramKey)
                : [queryParamMap.get(paramKey)]
            for (const paramValue of paramValues) {
                if (!paramValue) {
                    continue
                }

                let parsedValue = null
                switch (this.supportedQueryParamFilters()[paramKey].type) {
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
                        if (this.supportedQueryParamFilters()[paramKey].enumValues?.includes(paramValue)) {
                            parsedValue = paramValue
                            validFilters += 1
                            break
                        }

                        this.messageService.add({
                            severity: 'error',
                            summary: 'Wrong URL parameter value',
                            detail: `URL parameter ${paramKey} requires one of the values: ${this.supportedQueryParamFilters()[paramKey].enumValues.join(', ')}!`,
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
                            detail: `URL parameter ${paramKey} of type ${this.supportedQueryParamFilters()[paramKey].type} not supported!`,
                            life: 10000,
                        })
                        break
                }

                if (parsedValue !== null) {
                    const filterConstraint = {}
                    if (this.supportedQueryParamFilters()[paramKey].arrayType) {
                        parsedValue = _queryParamFilters[paramKey]?.value
                            ? [..._queryParamFilters[paramKey]?.value, parsedValue]
                            : [parsedValue]
                    }

                    filterConstraint[paramKey] = {
                        value: parsedValue,
                        matchMode: this.supportedQueryParamFilters()[paramKey].matchMode,
                    }
                    _queryParamFilters = { ..._queryParamFilters, ...filterConstraint }
                }
            }
        }

        return { count: validFilters, filters: _queryParamFilters }
    }

    private _filterTableByQueryParams(queryParamFilters: { [x: string]: FilterMetadata | FilterMetadata[] }): void {
        this.table()?.clearFilterValues()
        const metadata = this.table()?.createLazyLoadMetadata()
        this.table().filters = { ...metadata.filters, ...queryParamFilters }
        this.table()?._filter()
    }

    /**
     * Component lifecycle hook which inits the component.
     */
    ngOnInit(): void {
        console.log('storkTabViewComponent onInit')

        if (this.initWithEntitiesInCollection()) {
            this.entitiesCollection().forEach((entity: TEntity) => {
                this.openTabs.push(this.createTab(entity))
            })
            // this.openTabs = [...this.openTabs]
        }

        if (this.routePath()) {
            this.subscriptions = this.route.paramMap.subscribe({
                next: (params) => {
                    console.log('ActivatedRoute paramMap emits next', params, Date.now())
                    // const id = params.get('id')
                    // if (!id || id === this.firstTabRouteEnd()) {
                    //     console.log('open first tab')
                    //     this.activeTabEntityID = 0
                    //     return
                    // }
                    // const numericId = parseInt(id, 10)
                    // if (!Number.isNaN(numericId)) {
                    //     this.openTab(numericId)
                    //     return
                    // } else {
                    //     this.messageService.add({
                    //         detail: `Couldn't parse provided id ${id} to numeric value!`,
                    //         severity: 'error',
                    //         summary: `Error opening tab`,
                    //     })
                    //     this.activeTabEntityID = 0
                    //     return
                    // }
                },
                error: (err) => {
                    console.log('error emitted by ActivatedRoute paramMap', err)
                },
                complete: () => {
                    console.log('ActivatedRoute paramMap complete')
                },
            })
            this.subscriptions.add(
                combineLatest([this.route.queryParamMap, this.route.paramMap])
                    .pipe(
                        filter(() => this._initDone),
                        debounceTime(100)
                    )
                    // .pipe(filter(() => this._initDone))
                    .subscribe(
                        //     (value) => {
                        //     this._parseQueryParams(value)
                        //     this._filterZonesByQueryParams()
                        //     if (this.activeTabEntityID > 0) {
                        //         // Go back to first tab.
                        //         this.goToFirstTab()
                        //     }
                        // }
                        {
                            next: (combined) => {
                                console.log(
                                    'ActivatedRoute combined queryParamMap and paramMap emits next',
                                    combined,
                                    Date.now()
                                )
                                const queryParamMap = combined[0]
                                const paramMap = combined[1]
                                const id = paramMap.get('id')
                                if (!id || id === this.firstTabRouteEnd()) {
                                    console.log('open first tab')
                                    this.activeTabEntityID = 0
                                    if (this.table() && this.supportedQueryParamFilters()) {
                                        const f = this._parseQueryParams(queryParamMap)
                                        console.log('qParams filter parsing', f, this.table()?.filters)
                                        this._filterTableByQueryParams(f.filters)
                                    }
                                    return
                                }
                                const numericId = parseInt(id, 10)
                                if (!Number.isNaN(numericId)) {
                                    this.openTab(numericId)
                                    return
                                } else {
                                    this.messageService.add({
                                        detail: `Couldn't parse provided id ${id} to numeric value!`,
                                        severity: 'error',
                                        summary: `Error opening tab`,
                                    })
                                    this.activeTabEntityID = 0
                                    return
                                }
                            },
                            error: (err) => {
                                console.log('error emitted by ActivatedRoute queryParamMap', err)
                            },
                            complete: () => {
                                console.log('ActivatedRoute queryParamMap complete')
                            },
                        }
                    )
            )
            return
        }

        const filters = this._parseQueryParams(this.route.snapshot.queryParamMap)
        if (filters.count > 0) {
            // Valid filters found, so do not load lazily table data on init, because appropriate filters
            // will be applied later.
            this.loadTableDataOnInit.set(false)
            // this.dataLoading = true
        }

        if (this.openEntityID() > -1) {
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

    ngAfterViewInit() {
        this._initDone = true
        if (!this.loadTableDataOnInit()) {
            const filters = this._parseQueryParams(this.route.snapshot.queryParamMap)
            this._filterTableByQueryParams(filters.filters)
        }
    }

    /**
     * Debug string logger.
     * @param event
     */
    logChange(event: string | number) {
        console.log('storkTabViewComponent log onValueChange', event)
    }
}
