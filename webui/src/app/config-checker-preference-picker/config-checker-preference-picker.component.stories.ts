import { ConfigCheckerPreferencePickerComponent } from './config-checker-preference-picker.component'

import { StoryObj, Meta, moduleMetadata, applicationConfig } from '@storybook/angular'
import { ConfigChecker } from '../backend'
import { TableModule } from 'primeng/table'
import { ChipModule } from 'primeng/chip'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { ButtonModule } from 'primeng/button'
import { FormsModule } from '@angular/forms'
import { TriStateCheckboxModule } from 'primeng/tristatecheckbox'
import { TagModule } from 'primeng/tag'

export default {
    title: 'App/ConfigCheckerPreferencePicker',
    component: ConfigCheckerPreferencePickerComponent,
    decorators: [
        applicationConfig({
            providers: [],
        }),
        moduleMetadata({
            imports: [
                TableModule,
                ChipModule,
                OverlayPanelModule,
                BrowserAnimationsModule,
                ButtonModule,
                FormsModule,
                TriStateCheckboxModule,
                TagModule,
            ],
            declarations: [ConfigCheckerPreferencePickerComponent, HelpTipComponent],
        }),
    ],
    argTypes: {
        minimal: {
            defaultValue: false,
            type: 'boolean',
        },
        allowInheritState: {
            defaultValue: true,
            type: 'boolean',
        },
        loading: {
            defaultValue: false,
            type: 'boolean',
        },
        changePreferences: {
            action: 'onChangePreferences',
        },
    },
} as Meta

type Story = StoryObj<ConfigCheckerPreferencePickerComponent>

const mockData: ConfigChecker[] = [
    {
        name: 'dispensable_subnet',
        selectors: [
            'each-daemon',
            'kea-daemon',
            'kea-ca-daemon',
            'kea-dhcp-daemon',
            'kea-dhcp-v4-daemon',
            'kea-dhcp-v6-daemon',
            'kea-d2-daemon',
            'bind9-daemon',
            'unknown',
        ],
        state: ConfigChecker.StateEnum.Disabled,
        triggers: [
            'manual',
            'internal',
            'config change',
            'host reservations change',
            'Stork agent config change',
            'unknown',
        ],
        globallyEnabled: true,
    },
    {
        name: 'out_of_pool_reservation',
        selectors: ['each-daemon', 'kea-daemon'],
        state: ConfigChecker.StateEnum.Inherit,
        triggers: ['manual', 'config change'],
        globallyEnabled: false,
    },
]

export const Primary: Story = {
    args: {
        checkers: mockData,
    },
}

export const Empty: Story = {
    args: {
        checkers: [],
    },
}

export const Loading: Story = {
    args: {
        checkers: null,
        loading: true,
    },
}

export const Minimal: Story = {
    args: {
        minimal: true,
        checkers: mockData,
    },
}
