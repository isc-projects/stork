import { ConfigCheckerPreferencePickerComponent } from './config-checker-preference-picker.component'

import { Story, Meta, moduleMetadata, applicationConfig } from '@storybook/angular'
import { ConfigChecker } from '../backend'
import { TableModule } from 'primeng/table'
import { ChipModule } from 'primeng/chip'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { ButtonModule } from 'primeng/button'

export default {
    title: 'App/ConfigCheckerPreferencePicker',
    component: ConfigCheckerPreferencePickerComponent,
    decorators: [
        applicationConfig({
            providers: [],
        }),
        moduleMetadata({
            imports: [TableModule, ChipModule, OverlayPanelModule, BrowserAnimationsModule, ButtonModule],
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

const Template: Story<ConfigCheckerPreferencePickerComponent> = (args: ConfigCheckerPreferencePickerComponent) => ({
    props: args,
})

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

export const Primary = Template.bind({})
Primary.args = {
    checkers: mockData,
}

export const Empty = Template.bind({})
Empty.args = {
    checkers: [],
}

export const Loading = Template.bind({})
Loading.args = {
    checkers: null,
    loading: true,
}

export const Minimal = Template.bind({})
Minimal.args = {
    minimal: true,
    checkers: mockData,
}
