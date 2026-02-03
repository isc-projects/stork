import { Component, computed, contentChild, effect, inject, input, OnInit, signal, TemplateRef } from '@angular/core'
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
export class TableCaptionComponent implements OnInit {
    /**
     * Input PrimeNG table where this component is to be applied.
     */
    tableElement = input.required<Table>()

    /**
     * A key string to uniquely identify the table.
     */
    tableKey = input.required<string>()

    /**
     * A key for keeping filtering toolbar shown/hidden state in browser's local storage.
     */
    storageKey = computed(() => this.tableKey() + '-filters-toolbar-shown')

    /**
     * Input flag controlling whether the Filtering Help box should have wider or standard width.
     * Defaults to false.
     */
    wideHelpTip = input<boolean>(false)

    /**
     * Boolean flag controlling the filters toolbar shown/hidden state.
     */
    filtersShown = signal<boolean>(true)

    /**
     * Effect signal storing the filters toolbar shown/hidden state in local storage of the web browser.
     */
    filtersShownEffect = effect(() => this.storeFiltersShown(this.filtersShown()))

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
     * Initiates the component.
     */
    ngOnInit() {
        this.filtersShown.set(this.getFiltersShownFromStorage())
    }

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

    /**
     * Attempts to read shown/hidden state of the filtering toolbar from the browser's local storage.
     */
    getFiltersShownFromStorage(): boolean {
        const storage = localStorage.getItem(this.storageKey())
        if (!storage) {
            return true
        }

        const state = JSON.parse(storage)
        return state === true
    }

    /**
     * Attempts to store shown/hidden state of the filtering toolbar in the browser's local storage.
     * @param shown true if the toolbar is shown; false otherwise
     */
    storeFiltersShown(shown: boolean) {
        localStorage.setItem(this.storageKey(), JSON.stringify(shown))
    }
}
