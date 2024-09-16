import { ComponentFixture, TestBed } from '@angular/core/testing'

import { KeaGlobalConfigurationViewComponent } from './kea-global-configuration-view.component'
import { FieldsetModule } from 'primeng/fieldset'
import { CascadedParametersBoardComponent } from '../cascaded-parameters-board/cascaded-parameters-board.component'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { By } from '@angular/platform-browser'
import { ButtonModule } from 'primeng/button'
import { TableModule } from 'primeng/table'
import { TooltipModule } from 'primeng/tooltip'
import { DhcpOptionSetViewComponent } from '../dhcp-option-set-view/dhcp-option-set-view.component'
import { TreeModule } from 'primeng/tree'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TagModule } from 'primeng/tag'

describe('KeaGlobalConfigurationViewComponent', () => {
    let component: KeaGlobalConfigurationViewComponent
    let fixture: ComponentFixture<KeaGlobalConfigurationViewComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [
                ButtonModule,
                FieldsetModule,
                NoopAnimationsModule,
                TableModule,
                TooltipModule,
                TreeModule,
                OverlayPanelModule,
                TagModule,
            ],
            declarations: [
                CascadedParametersBoardComponent,
                KeaGlobalConfigurationViewComponent,
                DhcpOptionSetViewComponent,
                HelpTipComponent,
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(KeaGlobalConfigurationViewComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display configuration parameters view', () => {
        const fieldset = fixture.debugElement.query(By.css("[legend='Global DHCP Parameters']"))
        expect(fieldset).toBeTruthy()
        expect(fieldset.nativeElement.innerText).toContain('No parameters configured')
    })

    it('should display an edit button', () => {
        const button = fixture.debugElement.query(By.css('[label=Edit]'))
        expect(button).toBeTruthy()
    })

    it('should emit an event upon edit', () => {
        spyOn(component.editBegin, 'emit')
        component.onEditBegin()
        expect(component.editBegin.emit).toHaveBeenCalled()
    })

    it('should display the DHCP options view', () => {
        const fieldset = fixture.debugElement.query(By.css("[legend='Global DHCP Options']"))
        expect(fieldset).toBeTruthy()
        expect(fieldset.nativeElement.innerText).toContain('No options configured')
    })
})
