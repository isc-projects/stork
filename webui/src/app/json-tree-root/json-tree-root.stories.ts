import { JsonTreeRootComponent } from './json-tree-root.component'

import { StoryObj, Meta, moduleMetadata, applicationConfig } from '@storybook/angular'
import { Router, RouterModule } from '@angular/router'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { UsersService } from '../backend'
import { JsonTreeComponent } from '../json-tree/json-tree.component'
import { importProvidersFrom } from '@angular/core'

export default {
    title: 'App/JSON-Tree-Root',
    component: JsonTreeRootComponent,
    subcomponents: [JsonTreeComponent],
    decorators: [
        applicationConfig({
            providers: [
                MessageService,
                UsersService,
                {
                    provide: Router,
                    useValue: {
                        navigate: () => {},
                    },
                },
                importProvidersFrom(HttpClientTestingModule),
            ],
        }),
        moduleMetadata({
            imports: [HttpClientTestingModule, NoopAnimationsModule, RouterModule],
            declarations: [JsonTreeRootComponent, JsonTreeComponent],
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
