import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { UsersPageComponent } from './users-page.component'
import { ActivatedRoute, Router, RouterModule } from '@angular/router'
import { FormBuilder } from '@angular/forms'
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

    beforeEach(
        waitForAsync(() => {
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
                    FormBuilder,
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
        })
    )

    beforeEach(() => {
        fixture = TestBed.createComponent(UsersPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
