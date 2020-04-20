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
    @ViewChild(OverlayPanel, undefined)
    overlay: OverlayPanel

    /**
     * Title for the help box.
     */
    @Input() title: string

    /**
     * Width of the help box.
     */
    @Input() width: string = '20vw'

    constructor() {}

    ngOnInit() {}

    toggleOverlay(ev) {
        this.overlay.toggle(ev)
    }
}
