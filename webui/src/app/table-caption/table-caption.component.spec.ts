import { ComponentFixture, TestBed } from '@angular/core/testing'

import { TableCaptionComponent } from './table-caption.component'

describe('TableCaptionComponent', () => {
    let component: TableCaptionComponent
    let fixture: ComponentFixture<TableCaptionComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [TableCaptionComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(TableCaptionComponent)
        component = fixture.componentInstance
        fixture.componentRef.setInput('tableElement', {})
        fixture.componentRef.setInput('tableKey', 'key')
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should set storage key', () => {
        expect(component.storageKey()).toEqual('key-filters-toolbar-shown')
    })

    it('should support storing shown-hidden state to local storage', () => {
        // At first clear the storage.
        localStorage.clear()
        expect(localStorage.getItem(component.storageKey())).toBeFalsy()

        // It should be true by default.
        component.ngOnInit()
        expect(component.filtersShown()).toBeTrue()

        // Check if state is read from local storage.
        localStorage.setItem(component.storageKey(), JSON.stringify(false))
        component.ngOnInit()

        expect(component.getFiltersShownFromStorage()).toBeFalse()
        expect(component.filtersShown()).toBeFalse()

        // Check if state is stored in local storage.
        component.storeFiltersShown(true)

        expect(localStorage.getItem(component.storageKey())).toEqual('true')
        expect(component.getFiltersShownFromStorage()).toBeTrue()

        component.ngOnInit()
        expect(component.filtersShown()).toBeTrue()
    })
})
