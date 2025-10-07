import { ComponentFixture, TestBed } from '@angular/core/testing'

import { TabViewComponent } from './tab-view.component'
import { RouterModule } from '@angular/router'
import { MessageService } from 'primeng/api'
import { Component, viewChild } from '@angular/core'
import { Table, TableModule } from 'primeng/table'
import { Subnet } from '../backend'

describe('TabViewComponent', () => {
    let component: TabViewComponent<any, any>
    let fixture: ComponentFixture<TabViewComponent<any, any>>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [TabViewComponent, RouterModule.forRoot([]), TestComponent],
            providers: [MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(TabViewComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should compute first tab route', () => {
        expect(component.firstTabRoute()).toBeUndefined()

        fixture.componentRef.setInput('routePath', 'test')
        fixture.detectChanges()

        expect(component.routePath()).toEqual('test/') // check path sanitizer
        expect(component.firstTabRoute()).toBeTruthy()
        expect(component.firstTabRoute()).toEqual('test/all')

        fixture.componentRef.setInput('firstTabRouteEnd', 'custom')
        fixture.detectChanges()

        expect(component.firstTabRoute()).toBeTruthy()
        expect(component.firstTabRoute()).toEqual('test/custom')
    })

    it('should compute new entity tab route', () => {
        expect(component.newEntityTabRoute()).toBeUndefined()

        fixture.componentRef.setInput('routePath', 'test')
        fixture.detectChanges()

        expect(component.newEntityTabRoute()).toBeTruthy()
        expect(component.newEntityTabRoute()).toEqual('test/new')

        fixture.componentRef.setInput('newEntityTabRouteEnd', 'add')
        fixture.detectChanges()

        expect(component.newEntityTabRoute()).toBeTruthy()
        expect(component.newEntityTabRoute()).toEqual('test/add')
    })

    it('should compute entities collection', () => {
        const entities = [
            { id: 2, name: 'test' },
            { id: 3, name: 'test2' },
        ]
        fixture.componentRef.setInput('entities', entities)
        fixture.detectChanges()

        expect(component.entitiesCollection()).toBeTruthy()
        expect(component.entitiesCollection().length).toBe(2)
        expect(component.entitiesCollection()[0].id).toBe(2)
        expect(component.entitiesCollection()[0].name).toBe('test')
    })

    it('should init with all tabs open', () => {
        const entities = [
            { id: 7, name: 'test' },
            { id: 9, name: 'test2' },
        ]
        fixture.componentRef.setInput('entities', entities)
        fixture.componentRef.setInput('initWithEntitiesInCollection', true)
        spyOn(component.activeTabChange, 'emit')

        component.ngOnInit()
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(2)
        expect(component.activeTabEntityID).toBe(7)
        expect(component.activeTabChange.emit).toHaveBeenCalledWith(7)
    })

    it('should init with specific tab activated', () => {
        const entities = [
            { id: 7, name: 'test' },
            { id: 9, name: 'test2' },
        ]
        fixture.componentRef.setInput('entities', entities)
        fixture.componentRef.setInput('initWithEntitiesInCollection', true)
        fixture.componentRef.setInput('openEntityID', 9)

        component.ngOnInit()
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(2)
        expect(component.activeTabEntityID).toBe(9)
    })

    it('should close tab', () => {
        const entities = [
            { id: 7, name: 'test' },
            { id: 9, name: 'test2' },
        ]
        fixture.componentRef.setInput('entities', entities)
        fixture.componentRef.setInput('initWithEntitiesInCollection', true)
        fixture.componentRef.setInput('openEntityID', 9)

        component.ngOnInit()
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(2)
        expect(component.closableTabs()).toBeTrue()

        component.closeTab(9)
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(1)
        expect(component.activeTabEntityID).toBe(7)
    })

    it('should not close tab', () => {
        const entities = [
            { id: 7, name: 'test' },
            { id: 9, name: 'test2' },
        ]
        fixture.componentRef.setInput('entities', entities)
        fixture.componentRef.setInput('initWithEntitiesInCollection', true)

        component.ngOnInit()
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(2)
        expect(component.closableTabs()).toBeTrue()

        component.closeTab(11) // non existing tab
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(2)
    })

    it('should not close tab second', () => {
        const entities = [
            { id: 7, name: 'test' },
            { id: 9, name: 'test2' },
        ]
        fixture.componentRef.setInput('entities', entities)
        fixture.componentRef.setInput('initWithEntitiesInCollection', true)
        fixture.componentRef.setInput('closableTabs', false)

        component.ngOnInit()
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(2)

        component.closeTab(9)
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(2)
    })

    it('should open tab', () => {
        const entities = [
            { id: 7, name: 'test' },
            { id: 9, name: 'test2' },
        ]
        fixture.componentRef.setInput('entities', entities)

        component.ngOnInit()
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(0)

        component.openTab(7)
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(1)
        expect(component.activeTabEntityID).toBe(7)

        component.openTab(9)
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(2)
        expect(component.activeTabEntityID).toBe(9)

        component.closeTab(9)
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(1)

        component.openTab(9)
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(2)
        expect(component.activeTabEntityID).toBe(9)
    })

    it('should not open tab', () => {
        const entities = [
            { id: 7, name: 'test' },
            { id: 9, name: 'test2' },
        ]
        fixture.componentRef.setInput('entities', entities)

        component.ngOnInit()
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(0)

        component.openTab(6) // non existing entity
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(0)

        component.openTab(9)
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(1)
        expect(component.activeTabEntityID).toBe(9)

        component.openTab(9) // repeat the same
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(1)
        expect(component.activeTabEntityID).toBe(9)
    })
})

@Component({
    imports: [TabViewComponent, TableModule],
    template: ` <app-tab-view #tabs [entitiesTable]="table()">
        <ng-template #firstTab>
            <p-table [value]="entities"></p-table>
        </ng-template>
    </app-tab-view>`,
})
class TestComponent {
    entities: Subnet[] = []
    tabs = viewChild('tabs')
    table = viewChild(Table)
}
describe('TabViewTestComponent', () => {
    let testComponent: TestComponent
    let component: any
    let fixture: ComponentFixture<TestComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [TestComponent, RouterModule.forRoot([])],
            providers: [MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(TestComponent)
        testComponent = fixture.componentInstance
        testComponent.entities.push({
            id: 1,
            subnet: '1.0.0.0/32',
        } as Subnet)
        fixture.detectChanges()
        component = testComponent.tabs()
    })

    it('should create', () => {
        expect(testComponent).toBeTruthy()
    })

    it('should compute entities collection', () => {
        expect(component.entitiesCollection()).toBeTruthy()
        expect(component.entitiesCollection().length).toBeTruthy()
        expect(component.entitiesCollection()[0].id).toBe(1)
    })
})
