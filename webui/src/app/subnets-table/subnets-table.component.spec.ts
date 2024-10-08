import { ComponentFixture, fakeAsync, TestBed, tick } from '@angular/core/testing'

import { SubnetsTableComponent } from './subnets-table.component'
import { ButtonModule } from 'primeng/button'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { InputNumberModule } from 'primeng/inputnumber'
import { FormsModule } from '@angular/forms'
import { PanelModule } from 'primeng/panel'
import { MessageService } from 'primeng/api'
import { TableModule } from 'primeng/table'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { RouterModule } from '@angular/router'
import { SubnetsPageComponent } from '../subnets-page/subnets-page.component'
import { TagModule } from 'primeng/tag'
import { DropdownModule } from 'primeng/dropdown'
import { DHCPService, Subnets } from '../backend'
import { By } from '@angular/platform-browser'
import { of } from 'rxjs'
import { SubnetBarComponent } from '../subnet-bar/subnet-bar.component'
import { HumanCountPipe } from '../pipes/human-count.pipe'
import { HumanCountComponent } from '../human-count/human-count.component'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { TooltipModule } from 'primeng/tooltip'

describe('SubnetsTableComponent', () => {
    let component: SubnetsTableComponent
    let fixture: ComponentFixture<SubnetsTableComponent>
    let dhcpApi: DHCPService

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [MessageService],
            imports: [
                TableModule,
                HttpClientTestingModule,
                ButtonModule,
                OverlayPanelModule,
                InputNumberModule,
                FormsModule,
                PanelModule,
                BrowserAnimationsModule,
                TagModule,
                DropdownModule,
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
            ],
            declarations: [
                EntityLinkComponent,
                HelpTipComponent,
                HumanCountComponent,
                HumanCountPipe,
                SubnetBarComponent,
                SubnetsTableComponent,
                PluralizePipe,
            ],
        }).compileComponents()

        dhcpApi = TestBed.inject(DHCPService)
        fixture = TestBed.createComponent(SubnetsTableComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
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
                            id: 2,
                            machineHostname: 'foo',
                        },
                    ],
                },
            ],
            total: 3,
        }
        spyOn(dhcpApi, 'getSubnets').and.returnValue(of(subnets as any))
        component.grafanaUrl = 'http://localhost:3000/'
        component.loadData({
            first: 0,
            rows: 10,
        })
        tick()
        fixture.detectChanges()

        const grafanaIcons = fixture.debugElement.queryAll(By.css('.pi-chart-line'))
        expect(grafanaIcons?.length).toBe(3)
    }))
})
