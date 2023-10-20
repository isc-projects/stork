const config = {
    stories: ['../src/**/*.stories.mdx', '../src/**/*.stories.@(js|jsx|ts|tsx)'],

    addons: ['@storybook/addon-links', '@storybook/addon-interactions', 'storybook-addon-mock'],

    framework: {
        name: '@storybook/angular',
        options: {},
    },

    docs: {
        autodocs: false,
    },
}

export default config
