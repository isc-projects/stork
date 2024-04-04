import { HostDataSourceLabelComponent } from './host-data-source-label.component'

import { StoryObj, Meta } from '@storybook/angular'

export default {
    title: 'App/HostDataSourceLabel',
    component: HostDataSourceLabelComponent,
} as Meta

type Story = StoryObj<HostDataSourceLabelComponent>

export const Config: Story = {
    args: {
        dataSource: 'config',
    },
}

export const Api: Story = {
    args: {
        dataSource: 'api',
    },
}

export const Unknown: Story = {
    args: {
        dataSource: 'unknown',
    },
}
