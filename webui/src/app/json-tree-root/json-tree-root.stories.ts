import { JsonTreeRootComponent } from './json-tree-root.component'

import { Story, Meta, moduleMetadata } from '@storybook/angular'
import { Router, RouterModule } from '@angular/router'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { UsersService } from '../backend'
import { JsonTreeComponent } from '../json-tree/json-tree.component'

export default {
    title: 'App/JSON-Tree-Root',
    component: JsonTreeRootComponent,
    subcomponents: [JsonTreeComponent],
    decorators: [
        moduleMetadata({
            imports: [HttpClientTestingModule, NoopAnimationsModule, RouterModule],
            declarations: [JsonTreeRootComponent, JsonTreeComponent],
            providers: [
                MessageService,
                UsersService,
                {
                    provide: Router,
                    useValue: {
                        navigate: () => {},
                    },
                },
            ],
        }),
    ],
    argTypes: {
        value: { control: 'object' },
        customValueTemplates: { control: 'object', defaultValue: {} },
        secretKeys: { control: 'object', defaultValue: ['password', 'secret'] },
    },
} as Meta

const Template: Story<JsonTreeRootComponent> = (args: JsonTreeRootComponent) => ({
    props: args,
})

export const Basic = Template.bind({})

Basic.args = {
    value: 42,
}

export const Complex = Template.bind({})

Complex.args = {
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
}
