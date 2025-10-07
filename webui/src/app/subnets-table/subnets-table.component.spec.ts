import { ComponentFixture, fakeAsync, flush, TestBed, tick } from '@angular/core/testing'

import { SubnetsTableComponent } from './subnets-table.component'
import { ButtonModule } from 'primeng/button'
import { PopoverModule } from 'primeng/popover'
import { InputNumber, InputNumberModule } from 'primeng/inputnumber'
import { FormsModule } from '@angular/forms'
import { PanelModule } from 'primeng/panel'
import { MessageService } from 'primeng/api'
import { TableModule } from 'primeng/table'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { RouterModule } from '@angular/router'
import { SubnetsPageComponent } from '../subnets-page/subnets-page.component'
import { TagModule } from 'primeng/tag'
import { SelectModule } from 'primeng/select'
import { DHCPService, Subnets } from '../backend'
import { By } from '@angular/platform-browser'
import { of } from 'rxjs'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { HumanCountComponent } from '../human-count/human-count.component'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { TooltipModule } from 'primeng/tooltip'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { ManagedAccessDirective } from '../managed-access.directive'
import { UtilizationBarComponent } from '../utilization-bar/utilization-bar.component'
import { PoolBarsComponent } from '../pool-bars/pool-bars.component'
import { FloatLabelModule } from 'primeng/floatlabel'
import { OutOfPoolBarComponent } from '../out-of-pool-bar/out-of-pool-bar.component'
import { IconFieldModule } from 'primeng/iconfield'
import { InputIconModule } from 'primeng/inputicon'
import { FilterMetadata } from 'primeng/api/filtermetadata'

describe('SubnetsTableComponent', () => {
    let component: SubnetsTableComponent
    let fixture: ComponentFixture<SubnetsTableComponent>
    let dhcpApi: DHCPService

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
                EntityLinkComponent,
                HelpTipComponent,
                HumanCountComponent,
                HumanCountPipe,
                SubnetBarComponent,
                SubnetsTableComponent,
                PluralizePipe,
                UtilizationBarComponent,
                PoolBarsComponent,
                OutOfPoolBarComponent,
            ],
            imports: [
                TableModule,
                ButtonModule,
                PopoverModule,
                InputNumberModule,
                FormsModule,
                PanelModule,
                BrowserAnimationsModule,
                TagModule,
                SelectModule,
                RouterModule.forRoot([
                    {
                        path: 'dhcp/subnets',
                        pathMatch: 'full',
                        redirectTo: 'dhcp/subnets/all',
                    },
                    {
                        path: 'dhcp/subnets/:id',
                        component: SubnetsPageComponent,
                    },
                ]),
                TooltipModule,
                ManagedAccessDirective,
                FloatLabelModule,
                IconFieldModule,
                InputIconModule,
            ],
            providers: [MessageService, provideHttpClient(withInterceptorsFromDi()), provideHttpClientTesting()],
        }).compileComponents()

        dhcpApi = TestBed.inject(DHCPService)
        fixture = TestBed.createComponent(SubnetsTableComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
        // Do not save table state between tests, because that makes tests unstable.
        spyOn(component.table, 'saveState').and.callFake(() => {})
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display links to grafana', fakeAsync(() => {
        const subnets: Subnets = {
            items: [
                {
                    subnet: '192.0.2.0/24',
                    localSubnets: [
                        {
                            id: 1,
                            machineHostname: 'foo',
                        },
                    ],
                },
                {
                    subnet: '192.0.3.0/24',
                    localSubnets: [
                        {
                            id: 2,
                            machineHostname: 'foo',
                        },
                    ],
                },
                {
                    subnet: '2001:db8:1::/64',
                    localSubnets: [
                        {
                            id: 3,
                            machineHostname: 'foo',
                        },
                    ],
                },
            ],
            total: 3,
        }
        spyOn(dhcpApi, 'getSubnets').and.returnValue(of(subnets as any))
        component.grafanaUrl = 'http://localhost:3000/'
        component.grafanaDhcp4DashboardId = 'hRf18FvWz'
        component.grafanaDhcp6DashboardId = 'AQPHKJUGz'

        const loadMetadata = component.table.createLazyLoadMetadata()
        component.loadData(loadMetadata)
        tick()
        fixture.detectChanges()

        const grafanaIcons = fixture.debugElement.queryAll(By.css('a .pi-chart-line'))
        expect(grafanaIcons?.length).toBe(3)

        for (let grafanaIcon of grafanaIcons) {
            const parent = grafanaIcon.nativeElement.parentElement
            expect(parent.tagName).toBe('A')
            const href = parent.getAttribute('href')
            const title = parent.getAttribute('title')
            if (title.includes('subnet 3')) {
                expect(href).toContain('AQPHKJUGz')
            } else {
                expect(href).toContain('hRf18FvWz')
            }
        }
    }))

    it('should recognize that there are subnets with names', () => {
        // No subnets.
        component.dataCollection = []
        expect(component.isAnySubnetWithNameVisible).toBeFalse()

        // Subnet without names.
        component.dataCollection = [{ localSubnets: [{ userContext: {} }] }]
        expect(component.isAnySubnetWithNameVisible).toBeFalse()

        // Subnet with names.
        component.dataCollection = [
            {
                localSubnets: [
                    {
                        userContext: { 'subnet-name': 'foo' },
                    },
                ],
            },
        ]
        expect(component.isAnySubnetWithNameVisible).toBeTrue()

        // Subnet with names and without names.
        component.dataCollection = [
            {
                localSubnets: [{ userContext: { 'subnet-name': 'foo' } }],
            },
            {
                localSubnets: [{ userContext: {} }],
            },
        ]
        expect(component.isAnySubnetWithNameVisible).toBeTrue()

        // Subnet with multiple names.
        component.dataCollection = [
            {
                localSubnets: [{ userContext: { 'subnet-name': 'foo' } }],
            },
            {
                localSubnets: [{ userContext: { 'subnet-name': 'bar' } }],
            },
        ]
        expect(component.isAnySubnetWithNameVisible).toBeTrue()

        component.dataCollection = [
            {
                localSubnets: [{ userContext: {} }],
            },
            {
                localSubnets: [{ userContext: { 'subnet-name': 'bar' } }],
            },
        ]
        expect(component.isAnySubnetWithNameVisible).toBeTrue()
    })

    it('should recognize that there are subnets with multiple names', () => {
        // No local subnets.
        let subnet = {}
        expect(component.hasAssignedMultipleSubnetNames(subnet)).toBeFalse()

        // Subnet without names.
        subnet = { localSubnets: [{ userContext: {} }] }
        expect(component.hasAssignedMultipleSubnetNames(subnet)).toBeFalse()

        // Single local subnet.
        subnet = { localSubnets: [{ userContext: { 'subnet-name': 'foo' } }] }
        expect(component.hasAssignedMultipleSubnetNames(subnet)).toBeFalse()

        // Multiple local subnets with the same name.
        subnet = {
            localSubnets: [{ userContext: { 'subnet-name': 'foo' } }, { userContext: { 'subnet-name': 'foo' } }],
        }
        expect(component.hasAssignedMultipleSubnetNames(subnet)).toBeFalse()

        // Multiple subnets with different names.
        subnet = {
            localSubnets: [{ userContext: { 'subnet-name': 'foo' } }, { userContext: { 'subnet-name': 'bar' } }],
        }
        expect(component.hasAssignedMultipleSubnetNames(subnet)).toBeTrue()

        // Multiple subnets. One with a name, one without.
        subnet = { localSubnets: [{ userContext: { 'subnet-name': 'foo' } }, { userContext: {} }] }
        expect(component.hasAssignedMultipleSubnetNames(subnet)).toBeTrue()
    })

    it('should not filter the table by numeric input with value zero', fakeAsync(() => {
        // Arrange
        const getSubnetsSpy = spyOn(dhcpApi, 'getSubnets').and.returnValue(of({ items: [], total: 0 }) as any)
        const inputNumbers = fixture.debugElement.queryAll(By.directive(InputNumber))
        expect(inputNumbers).toBeTruthy()
        expect(inputNumbers.length).toEqual(2)
        spyOn(component, 'filterTable')

        // Act
        component.table.clear()
        tick(300)
        fixture.detectChanges()
        inputNumbers[0].componentInstance.handleOnInput(new InputEvent('input'), '', 0) // appId
        tick(300)
        fixture.detectChanges()
        inputNumbers[1].componentInstance.handleOnInput(new InputEvent('input'), '', 0) // subnetId
        tick(300)
        fixture.detectChanges()

        // Assert
        expect(getSubnetsSpy).toHaveBeenCalled()
        // Since zero is forbidden filter value for numeric inputs, we expect that minimum allowed value (i.e. 1) will be used.
        expect(component.filterTable).toHaveBeenCalledWith(1, component.table.filters['appId'] as FilterMetadata)
        expect(component.filterTable).toHaveBeenCalledWith(1, component.table.filters['subnetId'] as FilterMetadata)
        flush()
    }))
})
