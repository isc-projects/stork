import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { GlobalSearchComponent } from './global-search.component'
import { SearchService } from '../backend/api/api'
import { HttpClientTestingModule } from '@angular/common/http/testing'

describe('GlobalSearchComponent', () => {
    let component: GlobalSearchComponent
    let fixture: ComponentFixture<GlobalSearchComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            declarations: [GlobalSearchComponent],
            providers: [SearchService],
            imports: [HttpClientTestingModule],
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
