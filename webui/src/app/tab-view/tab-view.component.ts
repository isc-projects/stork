import {
    booleanAttribute,
    Component,
    computed,
    ContentChild,
    Input,
    OnDestroy,
    OnInit,
    TemplateRef,
} from '@angular/core'
import { Tab, TabList, TabPanel, TabPanels, Tabs } from 'primeng/tabs'
import { ActivatedRoute, Router, RouterLink } from '@angular/router'
import { inject, input } from '@angular/core'
import { Subscription } from 'rxjs'
import { MessageService } from 'primeng/api'
import { TimesIcon } from 'primeng/icons'
import { NgClass, NgTemplateOutlet } from '@angular/common'
import { getErrorMessage } from '../utils'

export type StorkTab = {
    title: string
    value: number
    route?: string | undefined
    icon?: string
    entity: { [key: string]: any }
}

/**
 *
 * @param value
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

@Component({
    selector: 'app-tab-view',
    standalone: true,
    imports: [Tabs, TabList, Tab, TabPanels, TabPanel, RouterLink, TimesIcon, NgTemplateOutlet, NgClass],
    templateUrl: './tab-view.component.html',
    styleUrl: './tab-view.component.sass',
})
export class TabViewComponent<TEntity> implements OnInit, OnDestroy {
    openTabs: StorkTab[] = []

    activeTabEntityID: number = 0

    closableTabs = input(true, { transform: booleanAttribute })

    initWithEntitiesInCollection = input(false, { transform: booleanAttribute })

    firstTabLabel = input('All')

    firstTabRouteEnd = input('all')

    routePath = input(undefined, { transform: sanitizePath })

    firstTabRoute = computed(() => (this.routePath() ? this.routePath() + this.firstTabRouteEnd() : undefined))

    entitiesCollection = input<TEntity[]>(undefined)

    entityIDKey = input('id')

    entityNameKey = input('name')

    entityRouteKey = input('route')

    entityTabIconKey = input('icon')

    @Input() entityProvider: (id: number) => Promise<TEntity>

    @ContentChild('firstTab', { descendants: false }) firstTabTemplate: TemplateRef<any> | undefined

    @ContentChild('entityTab', { descendants: false }) entityTabTemplate: TemplateRef<any> | undefined

    private readonly route = inject(ActivatedRoute)

    private readonly messageService = inject(MessageService)

    private readonly router = inject(Router)

    private subscriptions: Subscription

    /**
     *
     * @param id
     * @param title
     */
    onUpdateTabTitle = (id: number, title: string) => {
        console.log('onUpdateTabTitle', id, title)
        const existingTab = this.openTabs.find((tab) => tab.value === id)
        if (existingTab) {
            existingTab.title = title
        }
    }

    /**
     *
     * @param id
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

    /**
     *
     * @param entity
     */
    getID(entity: TEntity): number {
        return entity[this.entityIDKey()]
    }

    /**
     *
     * @param entity
     */
    getName(entity: TEntity): string {
        return entity[this.entityNameKey()]
    }

    /**
     *
     * @param entity
     * @private
     */
    private getIcon(entity: TEntity): string | undefined {
        return entity[this.entityTabIconKey()] || undefined
    }

    /**
     *
     * @param entityID
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
     *
     * @param entityID
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
     *
     * @param entity
     */
    createTab(entity: TEntity): StorkTab {
        return {
            title: this.getName(entity),
            value: this.getID(entity),
            entity: entity,
            route: this.routePath() ? this.routePath() + this.getID(entity) : undefined,
            icon: this.getIcon(entity),
        }
    }

    /**
     *
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
                    console.log('ActivatedRoute paramMap emits next', params)
                    const id = params.get('id')
                    if (!id || id === this.firstTabRouteEnd()) {
                        console.log('open first tab')
                        this.activeTabEntityID = 0
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
                    console.log('error emitted by ActivatedRoute paramMap', err)
                },
                complete: () => {
                    console.log('ActivatedRoute paramMap complete')
                },
            })
        } else {
            this.goToFirstTab()
        }
    }

    ngOnDestroy(): void {
        console.log('storkTabViewComponent onDestroy')
        this.subscriptions?.unsubscribe()
    }

    logChange($event) {
        console.log('storkTabViewComponent log onValueChange', $event)
    }
}
