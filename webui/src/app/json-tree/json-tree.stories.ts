import { JsonTreeComponent } from './json-tree.component'

import { Story, Meta, moduleMetadata } from '@storybook/angular'
import { PaginatorModule } from 'primeng/paginator'

export default {
    title: 'App/JSON-Tree',
    component: JsonTreeComponent,
    decorators: [
        moduleMetadata({
            imports: [PaginatorModule],
            declarations: [JsonTreeComponent],
        }),
    ],
    argTypes: {
        value: { control: 'object' },
        customValueTemplates: { defaultValue: {} },
        secretKeys: { control: 'object', defaultValue: ['password', 'secret'] },
    },
} as Meta

const Template: Story<JsonTreeComponent> = (args: JsonTreeComponent) => ({
    props: args,
})

export const Basic = Template.bind({})

Basic.args = {
    key: 'key',
    value: {
        foo: 42,
        bar: {
            a: 1,
            b: true,
            password: 'secret',
        },
    },
}

export const LongList = Template.bind({})
LongList.args = {
    key: 'key',
    value: {
        foo: [...Array(100).keys()],
    },
    forceOpenThisLevel: true,
}
