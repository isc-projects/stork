import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { UsersPageComponent } from './users-page.component'
import { ActivatedRoute, Router, RouterModule } from '@angular/router'
import { UntypedFormBuilder } from '@angular/forms'
import { ServicesService, UsersService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { of } from 'rxjs'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { TabMenuModule } from 'primeng/tabmenu'
import { MenuModule } from 'primeng/menu'
import { TableModule } from 'primeng/table'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { RouterTestingModule } from '@angular/router/testing'

class MockParamMap {
    get(name: string): string | null {
        return null
    }
}

describe('UsersPageComponent', () => {
    let component: UsersPageComponent
    let fixture: ComponentFixture<UsersPageComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [
                HttpClientTestingModule,
                TabMenuModule,
                MenuModule,
                TableModule,
                BreadcrumbModule,
                OverlayPanelModule,
                NoopAnimationsModule,
                RouterModule,
                RouterTestingModule,
            ],
            declarations: [UsersPageComponent, BreadcrumbsComponent, HelpTipComponent],
            providers: [
                UntypedFormBuilder,
                UsersService,
                ServicesService,
                MessageService,
                {
                    provide: ActivatedRoute,
                    useValue: {
                        paramMap: of(new MockParamMap()),
                    },
                },
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(UsersPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('Configuration')
        expect(breadcrumbsComponent.items[1].label).toEqual('Users')
    })
})
