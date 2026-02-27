const config = {
    stories: ['../src/**/*.stories.@(js|jsx|ts|tsx)'],

    addons: [// is this used?
    '@storybook/addon-links', '@storybook/addon-themes', 'storybook-addon-mock', '@storybook/addon-docs'],

    framework: '@storybook/angular',

    core: {
        disableTelemetry: true, // Disables telemetry
    },
}

export default config
