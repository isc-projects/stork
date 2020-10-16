import { async, ComponentFixture, TestBed } from '@angular/core/testing'
import { BreadcrumbsComponent } from './breadcrumbs.component'
import { RouterTestingModule } from '@angular/router/testing'

describe('BreadcrumbsComponent', () => {
    let component: BreadcrumbsComponent
    let fixture: ComponentFixture<BreadcrumbsComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [RouterTestingModule],
            declarations: [BreadcrumbsComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(BreadcrumbsComponent)
        component = fixture.componentInstance
        component.items = []
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
