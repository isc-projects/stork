import { Inject, Injectable } from '@angular/core'
import { DOCUMENT } from '@angular/common'

@Injectable({
    providedIn: 'root',
})
export class ThemeService {
    constructor(@Inject(DOCUMENT) private document: Document) {}

    /**
     *
     * @param theme
     */
    switchTheme(theme: string) {
        let themeLink = this.document.getElementById('stork-theme') as HTMLLinkElement

        if (themeLink) {
            themeLink.href = theme + '.css'
        }
    }
}
