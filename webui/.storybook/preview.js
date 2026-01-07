import { setCompodocJson } from '@storybook/addon-docs/angular'
import docJson from '../documentation.json'
import { applicationConfig, moduleMetadata } from '@storybook/angular'
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async'
import { providePrimeNG } from 'primeng/config'
import { ToastModule } from 'primeng/toast'
import AuraBluePreset from '../src/app/app.config'
import { withThemeByClassName } from '@storybook/addon-themes'
import { AuthService } from '../src/app/auth.service'
import { authServiceDecorator, MockedAuthService } from '../src/app/utils-stories'
setCompodocJson(docJson)

const preview = {
    globalTypes: {
        role: {
            description: 'User role',
            toolbar: {
                title: 'Role',
                items: ['super-admin', 'admin', 'read-only'],
                dynamicTitle: true,
            },
        },
    },
    initialGlobals: {
        role: 'super-admin',
    },
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
                // Use MockedAuthService in all stories so that user privileges may be controlled by global "User role" setting.
                { provide: AuthService, useClass: MockedAuthService },
            ],
        }),
        withThemeByClassName({
            themes: {
                light: 'light',
                dark: 'dark',
            },
            defaultTheme: 'light',
        }),
        authServiceDecorator,
    ],
}
export default preview
