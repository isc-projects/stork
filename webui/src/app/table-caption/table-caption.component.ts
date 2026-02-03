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

/**
 * This is a component that is supposed to be used in a Caption template of a PrimeNG table component.
 * It is meant to organize the table filters toolbar and buttons that usually are displayed above the table.
 * Very often it is a 'Refresh List' button or more.
 * The purpose of having this separate component, is to have the same filtering toolbar look and feel
 * in all Stork filterable tables.
 */
@Component({
    selector: 'app-table-caption',
    imports: [Toolbar, NgTemplateOutlet, Button, ToggleSwitch, FormsModule, HelpTipComponent],
    templateUrl: './table-caption.component.html',
    styleUrl: './table-caption.component.sass',
})
export class TableCaptionComponent {
    /**
     * Input PrimeNG table where this component is to be applied.
     */
    tableElement = input<Table>()

    /**
     * Input flag controlling whether the Filtering Help box should have wider or standard width.
     * Defaults to false.
     */
    wideHelpTip = input<boolean>(false)

    /**
     * Boolean flag controlling the filters toolbar shown/hidden state.
     */
    filtersShown: boolean = true

    /**
     * Defines the template for the text that should be displayed inside the help-tip for the filtering.
     */
    filteringHelpTip = contentChild<TemplateRef<any> | undefined>('helptip', { descendants: false })

    /**
     * Defines the template for the filters that should be displayed inside the filters' toolbar.
     */
    filters = contentChild<TemplateRef<any> | undefined>('filters', { descendants: false })

    /**
     * Defines the template for the buttons that should be displayed above the table.
     */
    buttons = contentChild<TemplateRef<any> | undefined>('buttons', { descendants: false })

    /**
     * Defines the template for the PrimeNG splitButton that should be displayed for narrower viewports instead of all buttons provided in `buttons` template.
     */
    splitButton = contentChild<TemplateRef<any> | undefined>('splitbutton', { descendants: false })

    /**
     * Angular router required for triggering the navigation.
     * @private
     */
    private router = inject(Router)

    /**
     * Reference to tableHasFilter function, so that it can be used in the HTML template.
     * @protected
     */
    protected readonly tableHasFilter = tableHasFilter

    /**
     * Clears the PrimeNG table filtering. As a result, table pagination is also reset.
     * It doesn't reset the table sorting, if any was applied.
     */
    clearTableFiltering() {
        this.tableElement()?.clearFilterValues()
        this.router.navigate([])
    }
}
