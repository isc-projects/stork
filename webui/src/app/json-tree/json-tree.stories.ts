import { JsonTreeComponent } from './json-tree.component'

import { StoryObj, Meta } from '@storybook/angular'

export default {
    title: 'App/JSON-Tree',
    component: JsonTreeComponent,
    decorators: [],
    argTypes: {
        value: { control: 'object' },
        customValueTemplates: { defaultValue: {} },
        secretKeys: { control: 'object', defaultValue: ['password', 'secret'] },
    },
} as Meta

type Story = StoryObj<JsonTreeComponent>

export const Basic: Story = {
    args: {
        key: 'key',
        value: {
            foo: 42,
            bar: {
                a: 1,
                b: true,
                password: 'secret',
            },
        },
    },
}

export const LongList: Story = {
    args: {
        key: 'key',
        value: {
            foo: [...Array(100).keys()],
        },
        forceOpenThisLevel: true,
    },
}
