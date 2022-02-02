import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { SharedNetworksPageComponent } from './shared-networks-page.component'
import { FormsModule } from '@angular/forms'
import { DropdownModule } from 'primeng/dropdown'
import { TableModule } from 'primeng/table'
import { TooltipModule } from 'primeng/tooltip'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { Router, ActivatedRoute, RouterModule } from '@angular/router'
import { DHCPService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { of } from 'rxjs'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { RouterTestingModule } from '@angular/router/testing'

describe('SharedNetworksPageComponent', () => {
    let component: SharedNetworksPageComponent
    let fixture: ComponentFixture<SharedNetworksPageComponent>
    let dhcpService: DHCPService

    beforeEach(
        waitForAsync(() => {
            TestBed.configureTestingModule({
                imports: [
                    FormsModule,
                    DropdownModule,
                    TableModule,
                    TooltipModule,
                    HttpClientTestingModule,
                    BreadcrumbModule,
                    OverlayPanelModule,
                    NoopAnimationsModule,
                    RouterModule,
                    RouterTestingModule,
                ],
                declarations: [SharedNetworksPageComponent, SubnetBarComponent, BreadcrumbsComponent, HelpTipComponent],
                providers: [DHCPService],
            })

            dhcpService = TestBed.inject(DHCPService)
        })
    )

    beforeEach(() => {
        const fakeResponses: any = [
            {
                items: [
                    {
                        name: 'frog',
                        subnets: [
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
                    },
                ],
                total: 10496,
            },
            {
                items: [
                    {
                        name: 'frog',
                        subnets: [
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
                                        statsCollectedAt: '1970-01-01T12:00:00.0Z',
                                    },
                                ],
                                pools: ['1.0.0.4-1.0.255.254'],
                                subnet: '1.0.0.0/16',
                            },
                        ],
                    },
                ],
                total: 10496,
            },
        ]
        spyOn(dhcpService, 'getSharedNetworks').and.returnValues(
            // The shared networks are fetched twice before the unit test starts.
            of(fakeResponses[0]),
            of(fakeResponses[0]),
            of(fakeResponses[1])
        )

        fixture = TestBed.createComponent(SharedNetworksPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should convert statistics to big integers', async () => {
        // Act
        await fixture.whenStable()

        // Assert
        const stats: { [key: string]: BigInt } = component.networks[0].subnets[0].localSubnets[0].stats
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

    it('should not fail on empty statistics', async () => {
        // Act
        component.loadNetworks({})
        await fixture.whenStable()

        // Assert
        expect(component.networks[0].subnets[0].localSubnets[0].stats).toBeUndefined()
        // No throw
    })
})
