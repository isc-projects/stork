import { EntityLinkComponent } from './entity-link.component'
import { applicationConfig, Meta, StoryObj } from '@storybook/angular'
import { provideRouter, withHashLocation } from '@angular/router'

const meta: Meta<EntityLinkComponent> = {
    title: 'App/EntityLinkComponent',
    component: EntityLinkComponent,
    args: {
        showEntityName: true,
        showIdentifier: false,
    },
    decorators: [
        applicationConfig({
            providers: [provideRouter([], withHashLocation())],
        }),
    ],
}

export default meta
type Story = StoryObj<EntityLinkComponent>

export const Defaults: Story = {
    args: {
        entity: 'custom entity',
    },
}
export const SharedNetwork: Story = {
    args: {
        entity: 'shared-network',
        attrs: { id: 1234, name: 'shared-network-name' },
    },
}

export const HostReservation: Story = {
    args: {
        entity: 'host',
        attrs: { id: 1234, label: 'host-reservation-name' },
    },
}

export const Subnet: Story = {
    args: {
        entity: 'subnet',
        attrs: { id: 1234, subnet: 'subnet-takes-precedence-over-prefix', prefix: 'subnet-prefix' },
    },
}

export const User: Story = {
    args: {
        entity: 'user',
        attrs: { id: 1234, login: 'login-takes-precedence-over-email', email: 'user@organization.org' },
    },
}

export const Machine: Story = {
    args: {
        entity: 'machine',
        attrs: {
            machineId: 9876,
            id: 1234,
            machineLabel: 'this-takes-precedence-over-label',
            label: 'this-takes-precedence-over-address',
            address: 'machine-address',
        },
    },
}

export const Daemon: Story = {
    args: {
        entity: 'daemon',
        attrs: {
            daemonId: 9876,
            id: 1234,
            daemonLabel: 'this-takes-precedence-over-name',
            name: 'dhcp4',
            label: 'label-is-last-in-fallback',
        },
    },
}
