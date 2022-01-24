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
    let dhcpService: DHCPService

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
                    NoopAnimationsModule,
                ],
                declarations: [SubnetsPageComponent, SubnetBarComponent, BreadcrumbsComponent, HelpTipComponent],
            })
            dhcpService = TestBed.inject(DHCPService)
        })
    )

    beforeEach(() => {
        const fakeResponse: any = {
            items: [
                {
                    clientClass: 'class-00-00',
                    id: 5,
                    localSubnets: [
                        {
                            appId: 27,
                            appName: 'kea@localhost',
                            id: 1,
                            machineAddress: 'localhost',
                            machineHostname: 'lv-pc',
                            stats: {
                                'assigned-addresses':
                                    '12345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                'total-addresses':
                                    '1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890',
                                'declined-addresses': '-2',
                            },
                            statsCollectedAt: '2022-01-19T12:10:22.513Z',
                        },
                    ],
                    pools: ['1.0.0.4-1.0.255.254'],
                    subnet: '1.0.0.0/16',
                },
            ],
            total: 10496,
        }
        spyOn(dhcpService, 'getSubnets').and.returnValue(of(fakeResponse))

        fixture = TestBed.createComponent(SubnetsPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should convert statistics to big integers', async () => {
        // Act
        component.loadSubnets({})
        await fixture.whenStable()

        // Assert
        const stats: { [key: string]: BigInt } = component.subnets[0].localSubnets[0].stats
        expect(stats['assigned-addresses']).toBe(
            BigInt('12345678901234567890123456789012345678901234567890123456789012345678901234567890')
        )
        expect(stats['total-addresses']).toBe(
            BigInt(
                '1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890'
            )
        )
        expect(stats['declined-addresses']).toBe(BigInt('-2'))
    })
})
