import { JsonTreeRootComponent } from './json-tree-root.component'

import { StoryObj, Meta, applicationConfig } from '@storybook/angular'
import { MessageService } from 'primeng/api'
import { JsonTreeComponent } from '../json-tree/json-tree.component'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

export default {
    title: 'App/JSON-Tree-Root',
    component: JsonTreeRootComponent,
    subcomponents: [JsonTreeComponent],
    decorators: [
        applicationConfig({
            providers: [MessageService, provideHttpClient(withInterceptorsFromDi())],
        }),
    ],
    argTypes: {
        value: { control: 'object' },
        customValueTemplates: { control: 'object', defaultValue: {} },
        secretKeys: { control: 'object', defaultValue: ['password', 'secret'] },
    },
} as Meta

type Story = StoryObj<JsonTreeRootComponent>

export const Basic: Story = {
    args: {
        value: 42,
    },
}

export const Complex: Story = {
    args: {
        value: {
            foo: {
                bar: {
                    baz: [
                        1,
                        2,
                        3,
                        4,
                        5,
                        {
                            a: true,
                            b: false,
                            secret: 'password',
                            password: 'secret',
                        },
                        7,
                        8,
                        9,
                    ],
                },
            },
        },
    },
}
