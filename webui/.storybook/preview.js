import { setCompodocJson } from '@storybook/addon-docs/angular'
import docJson from '../documentation.json'
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
}
export default preview
