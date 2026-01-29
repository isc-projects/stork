import { Component, contentChild, inject, input, TemplateRef } from '@angular/core'
import { Toolbar } from 'primeng/toolbar'
import { NgTemplateOutlet } from '@angular/common'
import { Button } from 'primeng/button'
import { tableHasFilter } from '../table'
import { Table } from 'primeng/table'
import { Router } from '@angular/router'
import { ToggleSwitch } from 'primeng/toggleswitch'
import { FormsModule } from '@angular/forms'
import { HelpTipComponent } from '../help-tip/help-tip.component'

@Component({
    selector: 'app-table-caption',
    imports: [Toolbar, NgTemplateOutlet, Button, ToggleSwitch, FormsModule, HelpTipComponent],
    templateUrl: './table-caption.component.html',
    styleUrl: './table-caption.component.sass',
})
export class TableCaptionComponent {
    filtersShown: boolean = true
    filteringHelpTip = contentChild<TemplateRef<any> | undefined>('helptip', { descendants: false })
    filters = contentChild<TemplateRef<any> | undefined>('filters', { descendants: false })
    buttons = contentChild<TemplateRef<any> | undefined>('buttons', { descendants: false })
    splitButton = contentChild<TemplateRef<any> | undefined>('splitbutton', { descendants: false })
    tableElement = input<Table>()
    protected readonly tableHasFilter = tableHasFilter
    router = inject(Router)
    wideHelpTip = input<boolean>(false)

    /**
     * Clears the PrimeNG table filtering. As a result, table pagination is also reset.
     * It doesn't reset the table sorting, if any was applied.
     */
    clearTableFiltering() {
        this.tableElement()?.clearFilterValues()
        this.router.navigate([])
    }
}
