import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { GlobalSearchComponent } from './global-search.component'
import { SearchService } from '../backend/api/api'
import { HttpClient, HttpHandler } from '@angular/common/http'

describe('GlobalSearchComponent', () => {
    let component: GlobalSearchComponent
    let fixture: ComponentFixture<GlobalSearchComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [GlobalSearchComponent],
            providers: [SearchService, HttpClient, HttpHandler],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(GlobalSearchComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
