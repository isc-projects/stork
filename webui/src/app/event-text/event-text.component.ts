import { Component, Input } from '@angular/core'

/**
 * A component that presents event text. It takes raw text, looks there for entities
 * in the form of tags like <app id="123" type="kea">, and translates this into text parts
 * that are rendered accordingly by template: either plain text or event-link is used
 * for given text part.
 */
@Component({
    selector: 'app-event-text',
    templateUrl: './event-text.component.html',
    styleUrls: ['./event-text.component.sass'],
})
export class EventTextComponent {
    private _text: string
    textParts = []

    @Input()
    set text(text: string) {
        this._text = text
        this.parseText()
    }

    constructor() {}

    /**
     * Parse event text and look there for entities in angle brackets (e.g. <app id="12">)
     * Store in textParts list of slices of the text and entity elements.
     */
    parseText() {
        // match e.g. <daemon id="123" name="dhcp4">
        const reEntity = /<(\w+) +((?:\w+="[^"]*" *){1,})>/g
        // match e.g. name="dhcp4"
        const reAttrs = /(\w+)="([^"]+)"/g
        const matches = this._text.matchAll(reEntity)

        let prevIdx = 0
        this.textParts = []

        // go through all matches and put to textParts slice with prev part of _text,
        // and found entity in match
        for (const m of matches) {
            this.textParts.push(['text', this._text.slice(prevIdx, m.index)])
            prevIdx = m.index + m[0].length

            const entity = m[1]
            const attrsTxt = m[2]
            const attrsMatches = attrsTxt.matchAll(reAttrs)
            const attrs = {}
            for (const am of attrsMatches) {
                attrs[am[1]] = am[2]
            }

            this.textParts.push([entity, attrs])
        }

        const lastPart = this._text.slice(prevIdx)
        if (lastPart.length > 0) {
            this.textParts.push(['text', lastPart])
        }
    }
}
