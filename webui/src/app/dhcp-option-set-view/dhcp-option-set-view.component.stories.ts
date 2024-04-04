import { moduleMetadata, Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { DhcpOptionSetViewComponent } from './dhcp-option-set-view.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TagModule } from 'primeng/tag'
import { TooltipModule } from 'primeng/tooltip'
import { TreeModule } from 'primeng/tree'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { CheckboxModule } from 'primeng/checkbox'
import { DividerModule } from 'primeng/divider'
import { FormsModule } from '@angular/forms'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'

export default {
    title: 'App/DhcpOptionSetView',
    component: DhcpOptionSetViewComponent,
    decorators: [
        applicationConfig({
            providers: [],
        }),
        moduleMetadata({
            imports: [
                CheckboxModule,
                DividerModule,
                FormsModule,
                NoopAnimationsModule,
                OverlayPanelModule,
                TagModule,
                TooltipModule,
                TreeModule,
            ],
            declarations: [DhcpOptionSetViewComponent, HelpTipComponent],
        }),
    ],
} as Meta

type Story = StoryObj<DhcpOptionSetViewComponent>

export const CombinedOptions: Story = {
    args: {
        levels: ['subnet', 'global'],
        options: [
            [
                {
                    alwaysSend: true,
                    code: 1024,
                    fields: [
                        {
                            fieldType: 'uint32',
                            values: ['111'],
                        },
                        {
                            fieldType: 'ipv6-prefix',
                            values: ['3000::', '64'],
                        },
                    ],
                    universe: 6,
                    options: [
                        {
                            code: 1025,
                            universe: 6,
                        },
                        {
                            code: 1026,
                            fields: [
                                {
                                    fieldType: 'ipv6-address',
                                    values: ['2001:db8:1::1'],
                                },
                                {
                                    fieldType: 'ipv6-address',
                                    values: ['2001:db8:2::1'],
                                },
                            ],
                            universe: 6,
                        },
                    ],
                },
                {
                    code: 1027,
                    fields: [
                        {
                            fieldType: 'bool',
                            values: ['true'],
                        },
                    ],
                    universe: 6,
                },
                {
                    code: 1028,
                    options: [
                        {
                            code: 1029,
                            fields: [
                                {
                                    fieldType: 'string',
                                    values: ['foo'],
                                },
                            ],
                            options: [
                                {
                                    code: 1030,
                                    options: [
                                        {
                                            code: 1031,
                                        },
                                    ],
                                },
                            ],
                            universe: 6,
                        },
                    ],
                    universe: 6,
                },
            ],
            [
                {
                    code: 1027,
                    fields: [
                        {
                            fieldType: 'bool',
                            values: ['false'],
                        },
                    ],
                    universe: 6,
                },
                {
                    code: 1030,
                    fields: [
                        {
                            fieldType: 'ipv4-address',
                            values: ['1.1.1.1'],
                        },
                    ],
                    universe: 6,
                },
            ],
        ],
    },
}
