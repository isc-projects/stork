import Aura from '@primeng/themes/aura'
import { definePreset } from '@primeng/themes'

// Generated API modules
import { Configuration, ConfigurationParameters } from './backend'

/** Create the OpenAPI client configuration. */
export function cfgFactory() {
    const params: ConfigurationParameters = {
        apiKeys: {},
        withCredentials: true,
    }
    return new Configuration(params)
}

const AuraBluePreset = definePreset(Aura, {
    semantic: {
        primary: {
            50: '{blue.50}',
            100: '{blue.100}',
            200: '{blue.200}',
            300: '{blue.300}',
            400: '{blue.400}',
            500: '{blue.500}',
            600: '{blue.600}',
            700: '{blue.700}',
            800: '{blue.800}',
            900: '{blue.900}',
            950: '{blue.950}',
        },
        colorScheme: {
            // Adding custom 'inverted' color scheme which mimics old PrimeNG 'surface' color scheme,
            // which for light scheme was changing from white to dark colors, and for
            // dark scheme it was changing from dark colors to white.
            // In new PrimeNG (v18 and following), the 'surface' color scheme behaves similarly for both light and dark mode,
            // i.e. it changes from white to darker colors.
            dark: {
                inverted: {
                    0: '{zinc.900}',
                    50: '{zinc.800}',
                    100: '{zinc.700}',
                    200: '{zinc.600}',
                    300: '{zinc.500}',
                    400: '{zinc.400}',
                    500: '{zinc.300}',
                    600: '{zinc.200}',
                    700: '{zinc.100}',
                    800: '{zinc.50}',
                    900: '#ffffff',
                    950: '#ffffff',
                },
            },
            light: {
                inverted: {
                    0: '#ffffff',
                    50: '{slate.50}',
                    100: '{slate.100}',
                    200: '{slate.200}',
                    300: '{slate.300}',
                    400: '{slate.400}',
                    500: '{slate.500}',
                    600: '{slate.600}',
                    700: '{slate.700}',
                    800: '{slate.800}',
                    900: '{slate.900}',
                    950: '{slate.950}',
                },
            },
        },
    },
    components: {
        // Apply primary background color for Chips instead of the default greyish surface color.
        chip: {
            colorScheme: {
                light: {
                    root: {
                        background: '{primary.100}',
                    },
                },
                dark: {
                    root: {
                        background: '{primary.400}',
                    },
                },
            },
        },
        // Make messages text lighter (500 by default).
        message: {
            colorScheme: {
                light: {
                    root: {
                        textFontWeight: '400',
                    },
                },
                dark: {
                    root: {
                        textFontWeight: '400',
                    },
                },
            },
        },
        // Apply regular padding for all panel headers.
        panel: {
            colorScheme: {
                light: {
                    root: {
                        toggleableHeaderPadding: '1.125rem',
                    },
                },
                dark: {
                    root: {
                        toggleableHeaderPadding: '1.125rem',
                    },
                },
            },
        },
        // Customize accordion header background colors and apply smaller padding for accordion panel content.
        accordion: {
            colorScheme: {
                light: {
                    root: {
                        headerBackground: '{surface.50}',
                        headerActiveBackground: '{surface.50}',
                        headerHoverBackground: '{surface.100}',
                        headerActiveHoverBackground: '{surface.100}',
                        contentPadding: '0 0.5rem 0.5rem 0.5rem',
                    },
                },
                dark: {
                    root: {
                        headerBackground: '{surface.950}',
                        headerActiveBackground: '{surface.950}',
                        headerHoverBackground: '{surface.800}',
                        headerActiveHoverBackground: '{surface.800}',
                        contentPadding: '0 0.5rem 0.5rem 0.5rem',
                    },
                },
            },
        },
    },
})

export default AuraBluePreset
