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
     * @param isDark
     */
    switchTheme(theme: string, isDark: boolean) {
        let themeLink = this.document.getElementById('stork-theme') as HTMLLinkElement
        this.document.body.classList.remove('dark', 'light')
        this.document.body.classList.add(isDark ? 'dark' : 'light')

        const darkLight = isDark ? 'dark' : 'light'
        if (themeLink) {
            themeLink.href = `${theme}-${darkLight}.css`
        }
    }
}
