import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { BreadcrumbsComponent } from './breadcrumbs.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { PopoverModule } from 'primeng/popover'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { provideRouter, RouterModule } from '@angular/router'

describe('BreadcrumbsComponent', () => {
    let component: BreadcrumbsComponent
    let fixture: ComponentFixture<BreadcrumbsComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [RouterModule, BreadcrumbModule, PopoverModule, NoopAnimationsModule],
            declarations: [BreadcrumbsComponent, HelpTipComponent],
            providers: [provideRouter([])],
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
