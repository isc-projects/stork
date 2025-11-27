import { Meta, StoryObj } from '@storybook/angular'
import { DelegatedPrefixBarComponent } from './delegated-prefix-bar.component'
import { DelegatedPrefixPool } from '../backend'

export default {
    title: 'App/DelegatedPrefixBar',
    component: DelegatedPrefixBarComponent,
    decorators: [],
} as Meta

type Story = StoryObj<DelegatedPrefixBarComponent>

export const StandardPrefix: Story = {
    args: {
        pool: {
            prefix: '3001:42::/64',
            delegatedLength: 80,
        } as DelegatedPrefixPool,
    },
}

export const ExcludedPrefix: Story = {
    args: {
        pool: {
            prefix: '2001:db8:1:8000::/48',
            delegatedLength: 64,
            excludedPrefix: '2001:db8:1:8000:cafe:80::/72',
        } as DelegatedPrefixPool,
    },
}
