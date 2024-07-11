import { ComponentFixture, TestBed } from '@angular/core/testing'

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

describe('SubnetsTableComponent', () => {
    let component: SubnetsTableComponent
    let fixture: ComponentFixture<SubnetsTableComponent>

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
            ],
            declarations: [SubnetsTableComponent, HelpTipComponent, PluralizePipe],
        }).compileComponents()

        fixture = TestBed.createComponent(SubnetsTableComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
