import { StoryObj, Meta, moduleMetadata, applicationConfig } from '@storybook/angular'
import { SharedNetworkFormComponent } from './shared-network-form.component'
import { toastDecorator } from '../utils-stories'
import { FieldsetModule } from 'primeng/fieldset'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { MessageService } from 'primeng/api'
import { ToastModule } from 'primeng/toast'
import { SharedParametersFormComponent } from '../shared-parameters-form/shared-parameters-form.component'
import { ButtonModule } from 'primeng/button'
import { CheckboxModule } from 'primeng/checkbox'
import { ChipsModule } from 'primeng/chips'
import { DropdownModule } from 'primeng/dropdown'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { InputNumberModule } from 'primeng/inputnumber'
import { TableModule } from 'primeng/table'
import { TagModule } from 'primeng/tag'
import { TriStateCheckboxModule } from 'primeng/tristatecheckbox'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { CreateSharedNetworkBeginResponse, UpdateSharedNetworkBeginResponse } from '../backend'
import { DhcpClientClassSetFormComponent } from '../dhcp-client-class-set-form/dhcp-client-class-set-form.component'
import { DhcpOptionSetFormComponent } from '../dhcp-option-set-form/dhcp-option-set-form.component'
import { DhcpOptionFormComponent } from '../dhcp-option-form/dhcp-option-form.component'
import { SplitButtonModule } from 'primeng/splitbutton'
import { DividerModule } from 'primeng/divider'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { RouterTestingModule } from '@angular/router/testing'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { MultiSelectModule } from 'primeng/multiselect'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { MessagesModule } from 'primeng/messages'
import { HttpClientModule } from '@angular/common/http'
import { importProvidersFrom } from '@angular/core'
import { AddressPoolFormComponent } from '../address-pool-form/address-pool-form.component'
import { AccordionModule } from 'primeng/accordion'
import { PrefixPoolFormComponent } from '../prefix-pool-form/prefix-pool-form.component'
import { ArrayValueSetFormComponent } from '../array-value-set-form/array-value-set-form.component'

let mockCreateSharedNetwork4BeginData: CreateSharedNetworkBeginResponse = {
    id: 123,
    daemons: [
        {
            id: 1,
            name: 'dhcp4',
            app: {
                name: 'first',
            },
            version: '2.7.0',
        },
        {
            id: 3,
            name: 'dhcp6',
            app: {
                name: 'first',
            },
            version: '2.7.0',
        },
        {
            id: 2,
            name: 'dhcp4',
            app: {
                name: 'second',
            },
            version: '2.4.0',
        },
        {
            id: 4,
            name: 'dhcp6',
            app: {
                name: 'second',
            },
            version: '2.7.0',
        },
        {
            id: 5,
            name: 'dhcp6',
            app: {
                name: 'third',
            },
            version: '2.7.0',
        },
    ],
    sharedNetworks4: ['floor1', 'floor2', 'floor3', 'stanza'],
    sharedNetworks6: [],
    clientClasses: ['foo', 'bar'],
}

let mockUpdateSharedNetwork4BeginData: UpdateSharedNetworkBeginResponse = {
    id: 123,
    sharedNetwork: {
        id: 123,
        name: 'stanza',
        universe: 4,
        localSharedNetworks: [
            {
                appId: 234,
                daemonId: 1,
                appName: 'server 1',
                keaConfigSharedNetworkParameters: {
                    sharedNetworkLevelParameters: {
                        allocator: 'random',
                        options: [
                            {
                                alwaysSend: true,
                                code: 5,
                                encapsulate: '',
                                fields: [
                                    {
                                        fieldType: 'ipv4-address',
                                        values: ['192.0.2.1'],
                                    },
                                ],
                                options: [],
                                universe: 4,
                            },
                        ],
                        optionsHash: '123',
                    },
                },
            },
            {
                appId: 234,
                daemonId: 2,
                appName: 'server 2',
                keaConfigSharedNetworkParameters: {
                    sharedNetworkLevelParameters: {
                        allocator: 'iterative',
                        options: [
                            {
                                alwaysSend: true,
                                code: 5,
                                encapsulate: '',
                                fields: [
                                    {
                                        fieldType: 'ipv4-address',
                                        values: ['192.0.2.2'],
                                    },
                                ],
                                options: [],
                                universe: 4,
                            },
                        ],
                        optionsHash: '234',
                    },
                },
            },
        ],
    },
    daemons: [
        {
            id: 1,
            name: 'dhcp4',
            app: {
                name: 'first',
            },
            version: '2.7.0',
        },
        {
            id: 3,
            name: 'dhcp6',
            app: {
                name: 'first',
            },
            version: '2.7.0',
        },
        {
            id: 2,
            name: 'dhcp4',
            app: {
                name: 'second',
            },
            version: '2.4.0',
        },
        {
            id: 4,
            name: 'dhcp6',
            app: {
                name: 'second',
            },
            version: '2.7.0',
        },
        {
            id: 5,
            name: 'dhcp6',
            app: {
                name: 'third',
            },
            version: '2.7.0',
        },
    ],
    sharedNetworks4: ['floor1', 'floor2', 'floor3', 'stanza'],
    sharedNetworks6: [],
    clientClasses: ['foo', 'bar'],
}

let mockUpdateSharedNetwork6BeginData: UpdateSharedNetworkBeginResponse = {
    id: 234,
    sharedNetwork: {
        id: 234,
        name: 'bella',
        universe: 6,
        localSharedNetworks: [
            {
                appId: 234,
                daemonId: 4,
                appName: 'server 1',
                keaConfigSharedNetworkParameters: {
                    sharedNetworkLevelParameters: {
                        allocator: 'random',
                        options: [
                            {
                                alwaysSend: true,
                                code: 23,
                                encapsulate: '',
                                fields: [
                                    {
                                        fieldType: 'ipv6-address',
                                        values: ['2001:db8:2::6789'],
                                    },
                                ],
                                options: [],
                                universe: 6,
                            },
                        ],
                        optionsHash: '123',
                    },
                },
            },
            {
                appId: 345,
                daemonId: 5,
                appName: 'server 2',
                keaConfigSharedNetworkParameters: {
                    sharedNetworkLevelParameters: {
                        allocator: 'random',
                        options: [
                            {
                                alwaysSend: true,
                                code: 23,
                                encapsulate: '',
                                fields: [
                                    {
                                        fieldType: 'ipv6-address',
                                        values: ['2001:db8:2::6789'],
                                    },
                                ],
                                options: [],
                                universe: 6,
                            },
                        ],
                        optionsHash: '123',
                    },
                },
            },
        ],
    },
    daemons: [
        {
            id: 1,
            name: 'dhcp4',
            app: {
                name: 'first',
            },
            version: '2.7.0',
        },
        {
            id: 3,
            name: 'dhcp6',
            app: {
                name: 'first',
            },
            version: '2.7.0',
        },
        {
            id: 2,
            name: 'dhcp4',
            app: {
                name: 'second',
            },
            version: '2.7.0',
        },
        {
            id: 4,
            name: 'dhcp6',
            app: {
                name: 'second',
            },
            version: '2.7.0',
        },
        {
            id: 5,
            name: 'dhcp6',
            app: {
                name: 'third',
            },
            version: '2.7.0',
        },
    ],
    sharedNetworks4: [],
    sharedNetworks6: ['floor1', 'floor2', 'floor3', 'bella'],
    clientClasses: ['foo', 'bar'],
}

export default {
    title: 'App/SharedNetworkForm',
    component: SharedNetworkFormComponent,
    decorators: [
        applicationConfig({
            providers: [
                MessageService,
                importProvidersFrom(HttpClientModule),
                importProvidersFrom(NoopAnimationsModule),
            ],
        }),
        moduleMetadata({
            imports: [
                AccordionModule,
                ButtonModule,
                CheckboxModule,
                ChipsModule,
                DividerModule,
                DropdownModule,
                FieldsetModule,
                FormsModule,
                HttpClientModule,
                InputNumberModule,
                MessagesModule,
                MultiSelectModule,
                TableModule,
                TagModule,
                TriStateCheckboxModule,
                OverlayPanelModule,
                ProgressSpinnerModule,
                ReactiveFormsModule,
                RouterTestingModule,
                SplitButtonModule,
                ToastModule,
            ],
            declarations: [
                AddressPoolFormComponent,
                ArrayValueSetFormComponent,
                DhcpClientClassSetFormComponent,
                DhcpOptionFormComponent,
                DhcpOptionSetFormComponent,
                EntityLinkComponent,
                HelpTipComponent,
                PrefixPoolFormComponent,
                SharedNetworkFormComponent,
                SharedParametersFormComponent,
            ],
        }),
        toastDecorator,
    ],
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/shared-networks/new/transaction',
                method: 'POST',
                status: 200,
                delay: 100,
                response: mockCreateSharedNetwork4BeginData,
            },
            {
                url: 'http://localhost/api/shared-networks/123/transaction',
                method: 'POST',
                status: 200,
                delay: 100,
                response: mockUpdateSharedNetwork4BeginData,
            },
            {
                url: 'http://localhost/api/shared-networks/234/transaction',
                method: 'POST',
                status: 200,
                delay: 100,
                response: mockUpdateSharedNetwork6BeginData,
            },
            {
                url: 'http://localhost/api/shared-networks/345/transaction',
                method: 'POST',
                status: 400,
                delay: 2000,
                response: mockUpdateSharedNetwork4BeginData,
            },
        ],
    },
} as Meta

type Story = StoryObj<SharedNetworkFormComponent>

export const NewSharedNetwork: Story = {
    args: {},
}

export const UpdatedSharedNetwork4: Story = {
    args: {
        sharedNetworkId: 123,
    },
}

export const UpdatedSharedNetwork6: Story = {
    args: {
        sharedNetworkId: 234,
    },
}

export const NoSharedNetworkId: Story = {
    args: {},
}

export const ErrorMessage: Story = {
    args: {
        sharedNetworkId: 345,
    },
}
