import { Component, Input, ViewChild } from '@angular/core'

import { Popover } from 'primeng/popover'

/**
 * Component displaying help box for widgets.
 *
 * It displays question mark icon. When this icon is clicked
 * then a help box is presented.
 */
@Component({
    selector: 'app-help-tip',
    templateUrl: './help-tip.component.html',
    styleUrls: ['./help-tip.component.sass'],
})
export class HelpTipComponent {
    /**
     * View child Popover component.
     */
    @ViewChild(Popover) popover: Popover

    /**
     * Target component to align the overlay of the help-tip.
     */
    @Input() target: any

    /**
     * Subject for which the help box is generated.
     */
    @Input({ required: true }) subject: string

    /**
     * Width of the help box.
     */
    @Input() width = '20vw'

    /**
     * A class for icon with question mark.
     */
    @Input() variant: 'big' | '' = ''

    /**
     * Custom button style.
     */
    helpTipButton = {
        root: {
            lgFontSize: '2rem',
        },
        colorScheme: {
            dark: {
                root: {
                    textPrimaryColor: '{primary.300}',
                    textPrimaryHoverBackground: '{primary.900}',
                },
            },
        },
    }

    constructor() {}

    /** Show/hide the help tip content. */
    toggleOverlay(ev) {
        this.popover.toggle(ev, this.target)
    }
}
