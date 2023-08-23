import { HostDataSourceLabelComponent } from './host-data-source-label.component'

import { Story, Meta } from '@storybook/angular'

export default {
    title: 'App/HostDataSourceLabel',
    component: HostDataSourceLabelComponent,
} as Meta

const Template: Story<HostDataSourceLabelComponent> = (args: HostDataSourceLabelComponent) => ({
    props: args,
})

export const Config = Template.bind({})

Config.args = {
    dataSource: 'config',
}

export const Api = Template.bind({})
Api.args = {
    dataSource: 'api',
}

export const Unknown = Template.bind({})
Unknown.args = {
    dataSource: 'unknown',
}
