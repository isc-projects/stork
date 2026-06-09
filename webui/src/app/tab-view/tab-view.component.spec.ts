import { ComponentFixture, TestBed } from '@angular/core/testing'

import { TabViewComponent } from './tab-view.component'
import { provideRouter } from '@angular/router'
import { MessageService } from 'primeng/api'
import { Component, viewChild } from '@angular/core'
import { Table, TableModule } from 'primeng/table'
import { Subnet } from '../backend'
describe('TabViewComponent', () => {
    let component: TabViewComponent<any, any>
    let fixture: ComponentFixture<TabViewComponent<any, any>>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [MessageService, provideRouter([])],
        }).compileComponents()

        fixture = TestBed.createComponent(TabViewComponent)
        component = fixture.componentInstance
    })

    it('should create', () => {
        fixture.detectChanges()
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

        fixture.detectChanges()

        expect(component.openTabs.length).toBe(2)
        expect(component.closableTabs()).toBeTrue()

        component.closeTab(9)

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

        fixture.detectChanges()

        expect(component.openTabs.length).toBe(2)
        expect(component.closableTabs()).toBeTrue()

        component.closeTab(11) // non existing tab

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

        fixture.detectChanges()

        expect(component.openTabs.length).toBe(2)

        component.closeTab(9)

        expect(component.openTabs.length).toBe(2)
    })

    it('should open tab', () => {
        const entities = [
            { id: 7, name: 'test' },
            { id: 9, name: 'test2' },
        ]
        fixture.componentRef.setInput('entities', entities)

        fixture.detectChanges()

        expect(component.openTabs.length).toBe(0)

        component.openTab(7)

        expect(component.openTabs.length).toBe(1)
        expect(component.activeTabEntityID).toBe(7)

        component.openTab(9)

        expect(component.openTabs.length).toBe(2)
        expect(component.activeTabEntityID).toBe(9)

        component.closeTab(9)

        expect(component.openTabs.length).toBe(1)

        component.openTab(9)

        expect(component.openTabs.length).toBe(2)
        expect(component.activeTabEntityID).toBe(9)
    })

    it('should not open tab', () => {
        const entities = [
            { id: 7, name: 'test' },
            { id: 9, name: 'test2' },
        ]
        fixture.componentRef.setInput('entities', entities)

        fixture.detectChanges()

        expect(component.openTabs.length).toBe(0)

        component.openTab(6) // non existing entity

        expect(component.openTabs.length).toBe(0)

        component.openTab(9)

        expect(component.openTabs.length).toBe(1)
        expect(component.activeTabEntityID).toBe(9)

        component.openTab(9) // repeat the same

        expect(component.openTabs.length).toBe(1)
        expect(component.activeTabEntityID).toBe(9)
    })

    it('should close tabs conditionally', () => {
        const entities = [
            { id: 6, name: 'test1' },
            { id: 7, name: 'test2' },
            { id: 8, name: 'test3' },
            { id: 9, name: 'test4' },
        ]
        fixture.componentRef.setInput('entities', entities)
        fixture.componentRef.setInput('initWithEntitiesInCollection', true)

        fixture.detectChanges()

        expect(component.openTabs.length).toBe(4)

        component.closeTabsConditionally((entity: { id: number; name: string }) => entity.id % 2 === 0) // close two tabs with even ids

        expect(component.openTabs.length).toBe(2)
        expect(component.openTabs[0].entity.name).toEqual('test2')
        expect(component.openTabs[1].entity.name).toEqual('test4')
        expect(component.activeTabEntityID).toBe(7)
    })

    it('should not delete entity that does not exist', () => {
        const entities = [
            { id: 2, name: 'test' },
            { id: 3, name: 'test2' },
        ]
        fixture.componentRef.setInput('entities', entities)
        fixture.detectChanges()

        component.onDeleteEntity(4)
        fixture.detectChanges()

        expect(component.entitiesCollection()).toBeTruthy()
        expect(component.entitiesCollection().length).toBe(2)
    })

    it('should not fail when trying to delete entity when entities collection is undefined', () => {
        component.onDeleteEntity(4)
        expect(component.entitiesCollection()).toBeFalsy()
    })

    it('should not update entity that does not exist', () => {
        const entities = [
            { id: 2, name: 'test' },
            { id: 3, name: 'test2' },
        ]
        fixture.componentRef.setInput('entities', entities)
        fixture.detectChanges()

        component.onUpdateTabEntity(4, { id: 4, name: 'updated' })
        fixture.detectChanges()

        expect(component.entitiesCollection()).toBeTruthy()
        expect(component.entitiesCollection().length).toBe(2)
        const updated = component.entitiesCollection().find((e) => e.id == 4)
        expect(updated).toBeFalsy()
    })

    it('should not fail when trying to update entity when entities collection is undefined', () => {
        component.onUpdateTabEntity(4)
        expect(component.entitiesCollection()).toBeFalsy()
    })
})

@Component({
    imports: [TabViewComponent, TableModule],
    template: ` <app-tab-view #tabs [entitiesTable]="table()" entityTitleKey="subnet">
        <ng-template #firstTab>
            <p-table [value]="entities">
                <ng-template #header>
                    <tr>
                        <th>ID</th>
                        <th>Subnet</th>
                    </tr>
                </ng-template>
                <ng-template #body let-subnet>
                    <tr>
                        <td>{{ subnet.id }}</td>
                        <td>{{ subnet.subnet }}</td>
                    </tr>
                </ng-template>
            </p-table>
        </ng-template>
        <ng-template #entityTab let-subnet>
            <div class="entity-id">{{ subnet.id }}</div>
            <div class="subnet">{{ subnet.subnet }}</div>
        </ng-template>
    </app-tab-view>`,
})
class TestComponent {
    entities: Subnet[] = []
    tabs = viewChild<TabViewComponent<Subnet, any>>('tabs')
    table = viewChild(Table)
}
describe('TabViewTestComponent', () => {
    let testComponent: TestComponent
    let component: TabViewComponent<Subnet, any>
    let fixture: ComponentFixture<TestComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [MessageService, provideRouter([])],
        }).compileComponents()

        fixture = TestBed.createComponent(TestComponent)
        testComponent = fixture.componentInstance
        testComponent.entities.push({
            id: 1,
            subnet: '1.0.0.0/32',
        } as Subnet)
        testComponent.entities.push({
            id: 3,
            subnet: '10.0.0.0/32',
        } as Subnet)
        fixture.detectChanges()
        component = testComponent.tabs()
    })

    it('should create', () => {
        expect(testComponent).toBeTruthy()
        expect(component).toBeTruthy()
    })

    it('should compute entities collection', () => {
        expect(component.entitiesCollection()).toBeTruthy()
        expect(component.entitiesCollection().length).toBeTruthy()
        expect(component.entitiesCollection()[0].id).toBe(1)
    })

    it('should update entity', () => {
        expect(component.entitiesCollection().length).toEqual(2)
        expect(component.openTabs.length).toEqual(0)

        component.openTab(1)
        component.openTab(3)

        expect(component.openTabs.length).toEqual(2)

        // Act - update the entity.
        component.onUpdateTabEntity(3, { id: 3, subnet: '10.0.0.0/31' })
        expect(component.getOpenTabEntity(3).subnet).toEqual('10.0.0.0/31')
        expect(component.entitiesCollection()[1].subnet).toEqual('10.0.0.0/31')
    })

    it('should update entity title', () => {
        expect(component.entitiesCollection().length).toEqual(2)
        expect(component.openTabs.length).toEqual(0)

        component.openTab(3)
        component.openTab(1)

        expect(component.openTabs.length).toEqual(2)

        // Act - update entity title.
        component.onUpdateTitle(1, '1.2.0.0/32')
        const tab = component.openTabs.find((t) => t.value === 1)
        expect(tab?.title).toEqual('1.2.0.0/32')
        expect(component.entitiesCollection()[0].subnet).toEqual('1.2.0.0/32')
    })

    it('should not update entity title in the table', () => {
        component.entityTitleProvider = (s) => `Title ${s.subnet}`

        expect(component.entitiesCollection().length).toEqual(2)
        expect(component.openTabs.length).toEqual(0)

        component.openTab(3)
        component.openTab(1)

        expect(component.openTabs.length).toEqual(2)

        // Act - update entity title.
        component.onUpdateTitle(1, '1.2.0.0/32')
        const tab = component.openTabs.find((t) => t.value === 1)
        expect(tab?.title).toContain('1.2.0.0/32')
        expect(component.entitiesCollection()[0].subnet)
            .withContext('entity title should not be updated in the table when entityTitleProvider is used')
            .toEqual('1.0.0.0/32')
    })
})
