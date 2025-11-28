const config = {
    stories: ['../src/**/*.mdx', '../src/**/*.stories.@(js|jsx|ts|tsx)'],

    addons: [
        '@storybook/addon-links', // is this used?
        '@storybook/addon-essentials',
        '@storybook/addon-interactions',
        '@storybook/addon-themes',
        'storybook-addon-mock',
    ],

    framework: {
        name: '@storybook/angular',
        options: {},
    },

    docs: {
        autodocs: false,
    },
}

export default config
