const config = {
    stories: ['../src/**/*.stories.@(js|jsx|ts|tsx)'],

    addons: [
        '@storybook/addon-links', // is this used?
        '@storybook/addon-essentials',
        '@storybook/addon-interactions',
        '@storybook/addon-themes',
        'storybook-addon-mock',
    ],

    framework:  '@storybook/angular',

    core: {
        disableTelemetry: true, // Disables telemetry
    },
}

export default config
