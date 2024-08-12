import { ComponentFixture, TestBed } from '@angular/core/testing'

import { SharedNetworksTableComponent } from './shared-networks-table.component'
import { MessageService } from 'primeng/api'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { TableModule } from 'primeng/table'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ButtonModule } from 'primeng/button'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { InputNumberModule } from 'primeng/inputnumber'
import { FormsModule } from '@angular/forms'
import { PanelModule } from 'primeng/panel'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { TagModule } from 'primeng/tag'
import { DropdownModule } from 'primeng/dropdown'
import { RouterModule } from '@angular/router'
import { SharedNetworksPageComponent } from '../shared-networks-page/shared-networks-page.component'

describe('SharedNetworksTableComponent', () => {
    let component: SharedNetworksTableComponent
    let fixture: ComponentFixture<SharedNetworksTableComponent>

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
                        path: 'dhcp/shared-networks',
                        pathMatch: 'full',
                        redirectTo: 'dhcp/shared-networks/all',
                    },
                    {
                        path: 'dhcp/shared-networks/:id',
                        component: SharedNetworksPageComponent,
                    },
                ]),
            ],
            declarations: [SharedNetworksTableComponent, HelpTipComponent, PluralizePipe],
        }).compileComponents()

        fixture = TestBed.createComponent(SharedNetworksTableComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
