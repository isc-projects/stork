import { setCompodocJson } from '@storybook/addon-docs/angular'
import docJson from '../documentation.json'
import { applicationConfig, moduleMetadata } from '@storybook/angular'
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async'
import { providePrimeNG } from 'primeng/config'
import { ToastModule } from 'primeng/toast'
import Aura from '@primeng/themes/aura'
import AuraBluePreset from '../src/app/app.config'
setCompodocJson(docJson)

const preview = {
    parameters: {
        controls: {
            matchers: {
                color: /(background|color)$/i,
            },
            exclude: /^_/,
        },

        // Disabled due to bug in Storybook for Angular 13
        // See: https://github.com/storybookjs/storybook/issues/17004
        // docs: { inlineStories: true },
        docs: false,
    },
    decorators: [
        moduleMetadata({
            // Import components injected by decorators.
            // The toastDecorator dependencies:
            imports: [ToastModule],
        }),
        applicationConfig({
            providers: [
                provideAnimationsAsync(),
                providePrimeNG({
                    theme: {
                        preset: AuraBluePreset,
                        options: {
                            darkModeSelector: '.dark',
                            cssLayer: {
                                name: 'primeng',
                                order: 'low, primeng, high',
                            },
                        },
                    },
                }),
            ],
        }),
    ],
}
export default preview
