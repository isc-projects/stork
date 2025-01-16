import { ComponentFixture, TestBed } from '@angular/core/testing'

import { MachinesTableComponent } from './machines-table.component'
import { RouterModule } from '@angular/router'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { ButtonModule } from 'primeng/button'
import { TableModule } from 'primeng/table'
import { PanelModule } from 'primeng/panel'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TriStateCheckboxModule } from 'primeng/tristatecheckbox'
import { FormsModule } from '@angular/forms'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { TagModule } from 'primeng/tag'

describe('MachinesTableComponent', () => {
    let component: MachinesTableComponent
    let fixture: ComponentFixture<MachinesTableComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [
                RouterModule.forRoot([]),
                HttpClientTestingModule,
                ButtonModule,
                TableModule,
                PanelModule,
                BrowserAnimationsModule,
                OverlayPanelModule,
                TriStateCheckboxModule,
                FormsModule,
                TagModule,
            ],
            declarations: [MachinesTableComponent, HelpTipComponent, PluralizePipe],
            providers: [MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(MachinesTableComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
