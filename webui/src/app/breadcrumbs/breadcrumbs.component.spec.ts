import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { BreadcrumbsComponent } from './breadcrumbs.component'
import { RouterTestingModule } from '@angular/router/testing'

describe('BreadcrumbsComponent', () => {
    let component: BreadcrumbsComponent
    let fixture: ComponentFixture<BreadcrumbsComponent>

    beforeEach(
        waitForAsync(() => {
            TestBed.configureTestingModule({
                imports: [RouterTestingModule],
                declarations: [BreadcrumbsComponent],
            }).compileComponents()
        })
    )

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
