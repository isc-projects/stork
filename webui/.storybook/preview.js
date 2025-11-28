import { setCompodocJson } from '@storybook/addon-docs/angular'
import docJson from '../documentation.json'
import { applicationConfig, componentWrapperDecorator, moduleMetadata } from '@storybook/angular'
import { provideAnimationsAsync } from '@angular/platform-browser/animations/async'
import { providePrimeNG } from 'primeng/config'
import { ToastModule } from 'primeng/toast'
import AuraBluePreset from '../src/app/app.config'
import { withThemeByClassName } from '@storybook/addon-themes'
import { AuthService } from '../src/app/auth.service'
import { ManagedAccessDirective } from '../src/app/managed-access.directive'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { MessageService } from 'primeng/api'
setCompodocJson(docJson)

let isSuperAdmin = true
let isAdmin = false
let isReadOnly = false

class MockedAuthService extends AuthService {
    superAdmin() {
        return isSuperAdmin
    }

    isAdmin() {
        return isAdmin
    }

    isInReadOnlyGroup() {
        return isReadOnly
    }
}

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
                provideHttpClient(withInterceptorsFromDi()),
                MessageService,
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
        moduleMetadata({
            imports: [ManagedAccessDirective],
        }),
        componentWrapperDecorator(
            (story) => story,
            ({ globals }) => {
                if (!globals.role) {
                    isSuperAdmin = true
                    isReadOnly = false
                    isAdmin = false
                    return
                }

                switch (globals.role) {
                    case 'super-admin':
                        isSuperAdmin = true
                        isAdmin = false
                        isReadOnly = false
                        break
                    case 'admin':
                        isSuperAdmin = false
                        isAdmin = true
                        isReadOnly = false
                        break
                    case 'read-only':
                        isReadOnly = true
                        isSuperAdmin = false
                        isAdmin = false
                        break
                    default:
                        isSuperAdmin = true
                        isReadOnly = false
                        isAdmin = false
                        break
                }
            }
        ),
    ],
}
export default preview
