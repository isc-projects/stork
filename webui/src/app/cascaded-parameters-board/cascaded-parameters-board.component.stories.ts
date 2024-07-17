import { moduleMetadata, Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { CascadedParametersBoardComponent } from './cascaded-parameters-board.component'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { TableModule } from 'primeng/table'
import { ButtonModule } from 'primeng/button'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { TooltipModule } from 'primeng/tooltip'
import { KeaConfigSubnetDerivedParameters } from '../backend'
import { TreeTableModule } from 'primeng/treetable'
import { ParameterViewComponent } from '../parameter-view/parameter-view.component'
import { UncamelPipe } from '../pipes/uncamel.pipe'
import { UnhyphenPipe } from '../pipes/unhyphen.pipe'

export default {
    title: 'App/CascadedParametersBoard',
    component: CascadedParametersBoardComponent,
    decorators: [
        applicationConfig({
            providers: [],
        }),
        moduleMetadata({
            imports: [ButtonModule, NoopAnimationsModule, TableModule, TooltipModule, TreeTableModule],
            declarations: [
                CascadedParametersBoardComponent,
                ParameterViewComponent,
                PlaceholderPipe,
                UncamelPipe,
                UnhyphenPipe,
            ],
            providers: [],
        }),
    ],
} as Meta

type Story = StoryObj<CascadedParametersBoardComponent<KeaConfigSubnetDerivedParameters>>

export const SameParameters: Story = {
    args: {
        levels: ['Subnet', 'Shared Network', 'Global'],
        data: [
            {
                name: 'Server1',
                parameters: [
                    {
                        cacheThreshold: 0.25,
                        cacheMaxAge: 1000,
                        clientClass: 'baz',
                        requireClientClasses: ['foo', 'bar'],
                        ddnsGeneratedPrefix: 'myhost',
                        ddnsOverrideClientUpdate: true,
                    },
                    {
                        cacheThreshold: 0.25,
                        cacheMaxAge: 1000,
                        clientClass: 'fbi',
                        requireClientClasses: ['abc'],
                        ddnsGeneratedPrefix: 'his',
                        ddnsOverrideClientUpdate: false,
                    },
                    {
                        cacheMaxAge: 1000,
                        requireClientClasses: ['abc'],
                        ddnsGeneratedPrefix: 'example',
                        ddnsOverrideClientUpdate: true,
                    },
                ],
            },
            {
                name: 'Server2',
                parameters: [
                    {
                        cacheThreshold: 0.22,
                        cacheMaxAge: 900,
                        clientClass: 'abc',
                        requireClientClasses: ['bar'],
                        ddnsGeneratedPrefix: 'hishost',
                        ddnsOverrideClientUpdate: true,
                    },
                    {
                        cacheThreshold: 0.21,
                        cacheMaxAge: 800,
                        clientClass: 'ibi',
                        requireClientClasses: ['abc', 'dec'],
                        ddnsGeneratedPrefix: 'her',
                        ddnsOverrideClientUpdate: true,
                    },
                    {
                        cacheMaxAge: 1000,
                        requireClientClasses: ['aaa'],
                        ddnsGeneratedPrefix: 'ours',
                        ddnsOverrideClientUpdate: false,
                    },
                ],
            },
        ],
    },
}

export const DistinctParameters: Story = {
    args: {
        levels: ['Subnet', 'Global'],
        data: [
            {
                name: 'Server1',
                parameters: [
                    {
                        cacheThreshold: 0.25,
                    },
                    {
                        cacheMaxAge: 1000,
                    },
                ],
            },
            {
                name: 'Server2',
                parameters: [
                    {
                        clientClass: 'abc',
                    },
                    {
                        requireClientClasses: ['abc', 'dec'],
                    },
                ],
            },
        ],
    },
}

export const ExcludedParameters: Story = {
    args: {
        levels: ['Subnet', 'Shared Network', 'Global'],
        data: [
            {
                name: 'Server1',
                parameters: [
                    {
                        cacheThreshold: 0.25,
                        cacheMaxAge: 1000,
                        clientClass: 'baz',
                        requireClientClasses: ['foo', 'bar'],
                        ddnsGeneratedPrefix: 'myhost',
                        ddnsOverrideClientUpdate: true,
                    },
                    {
                        cacheThreshold: 0.25,
                        cacheMaxAge: 1000,
                        clientClass: 'fbi',
                        requireClientClasses: ['abc'],
                        ddnsGeneratedPrefix: 'his',
                        ddnsOverrideClientUpdate: false,
                    },
                    {
                        cacheMaxAge: 1000,
                        requireClientClasses: ['abc'],
                        ddnsGeneratedPrefix: 'example',
                        ddnsOverrideClientUpdate: true,
                    },
                ],
            },
        ],
        excludedParameters: ['clientClass', 'ddnsOverrideClientUpdate'],
    },
}
