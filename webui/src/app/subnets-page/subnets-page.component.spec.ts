import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { SubnetsPageComponent } from './subnets-page.component'
import { FormsModule } from '@angular/forms'
import { DropdownModule } from 'primeng/dropdown'
import { TableModule } from 'primeng/table'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { TooltipModule } from 'primeng/tooltip'
import { RouterModule, ActivatedRoute, Router, convertToParamMap } from '@angular/router'
import { DHCPService, SettingsService, UsersService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { of } from 'rxjs'
import { MessageService } from 'primeng/api'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'

class MockParamMap {
    get(name: string): string | null {
        return null
    }
}

describe('SubnetsPageComponent', () => {
    let component: SubnetsPageComponent
    let fixture: ComponentFixture<SubnetsPageComponent>

    beforeEach(
        waitForAsync(() => {
            TestBed.configureTestingModule({
                providers: [
                    DHCPService,
                    UsersService,
                    MessageService,
                    SettingsService,
                    {
                        provide: ActivatedRoute,
                        useValue: {
                            snapshot: { queryParamMap: new MockParamMap() },
                            queryParamMap: of(new MockParamMap()),
                        },
                    },
                    {
                        provide: Router,
                        useValue: {},
                    },
                ],
                imports: [
                    FormsModule,
                    DropdownModule,
                    TableModule,
                    TooltipModule,
                    RouterModule,
                    HttpClientTestingModule,
                    BreadcrumbModule,
                    OverlayPanelModule,
                    NoopAnimationsModule
                ],
                declarations: [SubnetsPageComponent, SubnetBarComponent, BreadcrumbsComponent, HelpTipComponent],
            }).compileComponents()
        })
    )

    beforeEach(() => {
        fixture = TestBed.createComponent(SubnetsPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
