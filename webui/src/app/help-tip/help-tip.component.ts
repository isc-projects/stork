import { Component, OnInit, Input, ViewChild } from '@angular/core'

import { OverlayPanel } from 'primeng/overlaypanel'

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
export class HelpTipComponent implements OnInit {
    @ViewChild(OverlayPanel)
    overlay: OverlayPanel

    /**
     * Title for the help box.
     */
    @Input() title: string

    /**
     * Width of the help box.
     */
    @Input() width = '20vw'

    /**
     * A class for icon with question mark.
     */
    @Input() variant = ''

    constructor() {}

    ngOnInit() {}

    toggleOverlay(ev) {
        this.overlay.toggle(ev)
    }
}
