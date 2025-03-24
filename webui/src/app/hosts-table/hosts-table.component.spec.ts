import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { HostsTableComponent } from './hosts-table.component'
import { TableModule } from 'primeng/table'
import { RouterModule } from '@angular/router'
import { HostsPageComponent } from '../hosts-page/hosts-page.component'
import { MessageService } from 'primeng/api'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { ButtonModule } from 'primeng/button'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { InputNumberModule } from 'primeng/inputnumber'
import { FormsModule } from '@angular/forms'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { PanelModule } from 'primeng/panel'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { TagModule } from 'primeng/tag'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

describe('HostsTableComponent', () => {
    let component: HostsTableComponent
    let fixture: ComponentFixture<HostsTableComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            declarations: [HostsTableComponent, HelpTipComponent, PluralizePipe],
            imports: [
                TableModule,
                RouterModule.forRoot([
                    {
                        path: 'dhcp/hosts',
                        pathMatch: 'full',
                        redirectTo: 'dhcp/hosts/all',
                    },
                    {
                        path: 'dhcp/hosts/:id',
                        component: HostsPageComponent,
                    },
                ]),
                ButtonModule,
                OverlayPanelModule,
                InputNumberModule,
                FormsModule,
                PanelModule,
                BrowserAnimationsModule,
                TagModule,
            ],
            providers: [MessageService, provideHttpClient(withInterceptorsFromDi()), provideHttpClientTesting()],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HostsTableComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
