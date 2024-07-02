import { Inject, Injectable } from '@angular/core'
import { DOCUMENT } from '@angular/common'
import { BehaviorSubject } from 'rxjs'

/**
 * Service responsible for switching Stork UI theme.
 * Color theme and dark/light mode can be switched dynamically, without a need of refreshing the page.
 * Dark/light mode can be retrieved from OS/browser setting. User preference is stored in browser's storage.
 */
@Injectable({
    providedIn: 'root',
})
export class ThemeService {
    /**
     * Key used for storing user preference in browser's storage.
     * @private
     */
    private darkLightModeStorageKey = 'darkLightMode'

    /**
     * Boolean keeping state of dark mode enabled/disabled.
     * @private
     */
    private darkMode: boolean

    /**
     * Observable which may be used to notify subscribers about changed dark/light mode.
     */
    isDark$ = new BehaviorSubject<boolean>(false)

    /**
     * Service constructor.
     * @param document injected DOM document
     */
    constructor(@Inject(DOCUMENT) private document: Document) {}

    /**
     * Switches dark/light mode and color theme.
     * @param isDark when true provided, dark mode is enabled; otherwise light mode is enabled
     * @param theme color theme which may be also switched; defaults to PrimeNGv17 default `aura-blue` theme
     */
    switchTheme(isDark: boolean, theme: string = 'aura-blue') {
        this.darkMode = isDark
        const darkLight = isDark ? 'dark' : 'light'
        const themeLink = this.document.getElementById('stork-theme') as HTMLLinkElement

        // Store dark/light mode also as a class in <body> element.
        // This is needed for custom styling of some components that may differ for light and dark mode.
        this.document.body.classList.remove('dark', 'light')
        this.document.body.classList.add(darkLight)

        // Update dynamically Stork theme.
        if (themeLink) {
            themeLink.href = `${theme}-${darkLight}.css`
        }

        // Notify subscribers about dark/light mode switch.
        this.isDark$.next(this.darkMode)
    }

    /**
     * Sets initial dark/light mode theme.
     * Takes into account user stored preference and os/browser settings.
     */
    setInitialTheme() {
        // Let's retrieve OS/browser dark/light mode preference.
        const systemMode = window.matchMedia('(prefers-color-scheme: dark)')

        // User preferred mode should be stored in browser's local storage, it takes precedence over systemMode.
        const storedMode = localStorage.getItem(this.darkLightModeStorageKey)

        if (storedMode && (storedMode === 'dark' || storedMode === 'light')) {
            // If storedMode is found, apply the mode and return.
            this.switchTheme(storedMode === 'dark')
            return
        }

        // In case no storedMode was found, let's try to apply systemMode.
        if (systemMode && 'matches' in systemMode) {
            this.switchTheme(systemMode.matches === true)
            return
        }

        // In case none of above worked, apply default light mode.
        this.switchTheme(false)
    }

    /**
     * Stores user's preferred dark/light mode in browser's local storage.
     */
    storeTheme(): void {
        localStorage.setItem(this.darkLightModeStorageKey, this.darkMode ? 'dark' : 'light')
    }
}
