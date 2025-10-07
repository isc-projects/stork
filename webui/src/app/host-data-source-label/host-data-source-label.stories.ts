import { HostDataSourceLabelComponent } from './host-data-source-label.component'

import { StoryObj, Meta, moduleMetadata } from '@storybook/angular'
import { TagModule } from 'primeng/tag'

export default {
    title: 'App/HostDataSourceLabel',
    component: HostDataSourceLabelComponent,
    decorators: [
        moduleMetadata({
            imports: [TagModule],
        }),
    ],
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
