import { ComponentFixture, TestBed } from '@angular/core/testing'

import { TabViewComponent } from './tab-view.component'
import { RouterModule } from '@angular/router'
import { MessageService } from 'primeng/api'
import { Component, viewChild } from '@angular/core'
import { Table, TableModule } from 'primeng/table'
import { Subnet } from '../backend'
import { By } from '@angular/platform-browser'

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

    it('should close tabs conditionally', () => {
        const entities = [
            { id: 6, name: 'test1' },
            { id: 7, name: 'test2' },
            { id: 8, name: 'test3' },
            { id: 9, name: 'test4' },
        ]
        fixture.componentRef.setInput('entities', entities)
        fixture.componentRef.setInput('initWithEntitiesInCollection', true)

        component.ngOnInit()
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(4)

        component.closeTabsConditionally((entity: { id: number; name: string }) => entity.id % 2 === 0) // close two tabs with even ids
        fixture.detectChanges()

        expect(component.openTabs.length).toBe(2)
        expect(component.openTabs[0].entity.name).toEqual('test2')
        expect(component.openTabs[1].entity.name).toEqual('test4')
        expect(component.activeTabEntityID).toBe(7)
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
            imports: [TestComponent, RouterModule.forRoot([])],
            providers: [MessageService],
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
        fixture.detectChanges()
        component.openTab(3)
        fixture.detectChanges()

        expect(component.openTabs.length).toEqual(2)

        // Check displayed tab content.
        const id = fixture.debugElement.query(By.css('.entity-id'))
        expect(id).toBeTruthy()
        expect(id.nativeElement.innerText).toEqual('3')

        const subnet = fixture.debugElement.query(By.css('.subnet'))
        expect(subnet).toBeTruthy()
        expect(subnet.nativeElement.innerText).toEqual('10.0.0.0/32')

        // Check active tab title.
        const title = fixture.debugElement.query(By.css('.p-tab-active')) // PrimeNG selector used!! might change in the future!
        expect(title).toBeTruthy()
        expect(title.nativeElement.innerText).toEqual('10.0.0.0/32')

        // Act - update the entity.
        component.onUpdateTabEntity(3, { id: 3, subnet: '10.0.0.0/31' })
        fixture.detectChanges()

        // Check if displayed tab content was updated.
        const updatedSubnet = fixture.debugElement.query(By.css('.subnet'))
        expect(updatedSubnet).toBeTruthy()
        expect(updatedSubnet.nativeElement.innerText).toEqual('10.0.0.0/31')

        // Check if active tab title was updated.
        const updatedTitle = fixture.debugElement.query(By.css('.p-tab-active')) // PrimeNG selector used!! might change in the future!
        expect(updatedTitle).toBeTruthy()
        expect(updatedTitle.nativeElement.innerText).toEqual('10.0.0.0/31')

        // Go to the first tab with the table.
        component.activeTabEntityID = 0
        fixture.detectChanges()

        // Check if table content was updated.
        const tds = fixture.debugElement.queryAll(By.css('td'))
        expect(tds).toBeTruthy()
        expect(tds.length).toBe(4)
        expect(tds[0].nativeElement.innerText).toEqual('1')
        expect(tds[1].nativeElement.innerText).toEqual('1.0.0.0/32')
        expect(tds[2].nativeElement.innerText).toEqual('3')
        expect(tds[3].nativeElement.innerText).toEqual('10.0.0.0/31') // this should be also updated
    })

    it('should update entity title', () => {
        expect(component.entitiesCollection().length).toEqual(2)
        expect(component.openTabs.length).toEqual(0)

        component.openTab(3)
        fixture.detectChanges()
        component.openTab(1)
        fixture.detectChanges()

        expect(component.openTabs.length).toEqual(2)

        // Check displayed tab content.
        const id = fixture.debugElement.query(By.css('.entity-id'))
        expect(id).toBeTruthy()
        expect(id.nativeElement.innerText).toEqual('1')

        const subnet = fixture.debugElement.query(By.css('.subnet'))
        expect(subnet).toBeTruthy()
        expect(subnet.nativeElement.innerText).toEqual('1.0.0.0/32')

        // Check active tab title.
        const title = fixture.debugElement.query(By.css('.p-tab-active')) // PrimeNG selector used!! might change in the future!
        expect(title).toBeTruthy()
        expect(title.nativeElement.innerText).toEqual('1.0.0.0/32')

        // Act - update entity title.
        component.onUpdateTabTitle(1, '1.2.0.0/32')
        fixture.detectChanges()

        // Check if active tab title was updated.
        const updatedTitle = fixture.debugElement.query(By.css('.p-tab-active')) // PrimeNG selector used!! might change in the future!
        expect(updatedTitle).toBeTruthy()
        expect(updatedTitle.nativeElement.innerText).toEqual('1.2.0.0/32')

        // Go to the first tab with the table.
        component.activeTabEntityID = 0
        fixture.detectChanges()

        // Check if table content was updated.
        const tds = fixture.debugElement.queryAll(By.css('td'))
        expect(tds).toBeTruthy()
        expect(tds.length).toBe(4)
        expect(tds[0].nativeElement.innerText).toEqual('1')
        expect(tds[1].nativeElement.innerText)
            .withContext('entity title should be also updated in the table')
            .toEqual('1.2.0.0/32') // this should be also updated
        expect(tds[2].nativeElement.innerText).toEqual('3')
        expect(tds[3].nativeElement.innerText).toEqual('10.0.0.0/32')
    })

    it('should not update entity title in the table', () => {
        component.entityTitleProvider = (s) => `Title ${s.subnet}`

        expect(component.entitiesCollection().length).toEqual(2)
        expect(component.openTabs.length).toEqual(0)

        component.openTab(3)
        fixture.detectChanges()
        component.openTab(1)
        fixture.detectChanges()

        expect(component.openTabs.length).toEqual(2)

        // Check displayed tab content.
        const id = fixture.debugElement.query(By.css('.entity-id'))
        expect(id).toBeTruthy()
        expect(id.nativeElement.innerText).toEqual('1')

        const subnet = fixture.debugElement.query(By.css('.subnet'))
        expect(subnet).toBeTruthy()
        expect(subnet.nativeElement.innerText).toEqual('1.0.0.0/32')

        // Check active tab title.
        const title = fixture.debugElement.query(By.css('.p-tab-active')) // PrimeNG selector used!! might change in the future!
        expect(title).toBeTruthy()
        expect(title.nativeElement.innerText).toEqual('Title 1.0.0.0/32')

        // Act - update entity title.
        component.onUpdateTabTitle(1, '1.2.0.0/32')
        fixture.detectChanges()

        // Check if active tab title was updated.
        const updatedTitle = fixture.debugElement.query(By.css('.p-tab-active')) // PrimeNG selector used!! might change in the future!
        expect(updatedTitle).toBeTruthy()
        expect(updatedTitle.nativeElement.innerText).withContext('').toEqual('1.2.0.0/32')

        // Go to the first tab with the table.
        component.activeTabEntityID = 0
        fixture.detectChanges()

        // Check if table content is not touched.
        const tds = fixture.debugElement.queryAll(By.css('td'))
        expect(tds).toBeTruthy()
        expect(tds.length).toBe(4)
        expect(tds[0].nativeElement.innerText).toEqual('1')
        expect(tds[1].nativeElement.innerText)
            .withContext('entity title should not be updated in the table when entityTitleProvider is used')
            .toEqual('1.0.0.0/32') // this should not be not updated
        expect(tds[2].nativeElement.innerText).toEqual('3')
        expect(tds[3].nativeElement.innerText).toEqual('10.0.0.0/32')
    })
})
