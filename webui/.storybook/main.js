module.exports = {
    stories: ['../src/**/*.stories.mdx', '../src/**/*.stories.@(js|jsx|ts|tsx)'],

    addons: [
        '@storybook/addon-links',
        {
            name: '@storybook/addon-essentials',
            options: {
                docs: false,
            },
        },
        '@storybook/addon-interactions',
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
