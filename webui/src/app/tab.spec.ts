import { TestBed } from '@angular/core/testing'
import { Tab, TabType } from './tab'

class TestState {}

describe('tab', () => {
    beforeEach(() => TestBed.configureTestingModule({}))

    it('should create display tab', () => {
        let tab = new Tab<TestState, number>(TestState, TabType.Display, 5)
        expect(tab.state).toBeNull()
        expect(tab.tabSubject).toBe(5)
        expect(tab.tabType).toBe(TabType.Display)
    })

    it('should create new tab', () => {
        let tab = new Tab<TestState, number>(TestState, TabType.New, 5)
        expect(tab.state).not.toBeNull()
        expect(tab.state).toBeInstanceOf(TestState)
        expect(tab.tabSubject).toBe(5)
        expect(tab.tabType).toBe(TabType.New)
    })

    it('should create edit tab', () => {
        let tab = new Tab<TestState, number>(TestState, TabType.Edit, 5)
        expect(tab.state).not.toBeNull()
        expect(tab.state).toBeInstanceOf(TestState)
        expect(tab.tabSubject).toBe(5)
        expect(tab.tabType).toBe(TabType.Edit)
    })

    it('should set tab type', () => {
        let tab = new Tab<TestState, number>(TestState, TabType.Edit, 5)
        tab.setTabType(TabType.Display)
        expect(tab.state).toBeNull()
        expect(tab.tabType).toBe(TabType.Display)
    })
})
