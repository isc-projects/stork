import { IdentifierComponent } from './identifier.component'

import { StoryObj, Meta, applicationConfig } from '@storybook/angular'
import { provideRouter, withHashLocation } from '@angular/router'

export default {
    title: 'App/Identifier',
    component: IdentifierComponent,
    decorators: [
        applicationConfig({
            providers: [provideRouter([{ path: '**', component: IdentifierComponent }], withHashLocation())],
        }),
    ],
} as Meta

type Story = StoryObj<IdentifierComponent>

export const NoLabel: Story = {
    args: {
        hexValue: '73:30:6d:45:56:61:4c:75:65',
    },
}

export const WithLabel: Story = {
    args: {
        hexValue: '73:30:6d:45:56:61:4c:75:65',
        label: 'flex-id',
    },
}

export const WithLink: Story = {
    args: {
        hexValue: '73:30:6d:45:56:61:4c:75:65',
        link: 'foo/bar',
    },
}

export const NonPrintableCharacters: Story = {
    args: {
        hexValue: '00:10:B0:7F:80:D4:FF',
        label: 'flex-id',
    },
}

export const InvalidHexValue: Story = {
    args: {
        hexValue: 'It is not a hex value',
    },
}

export const Empty: Story = {
    args: {
        hexValue: '    ',
    },
}

export const EmptyWithLabel: Story = {
    args: {
        hexValue: '    ',
        label: 'flex-id',
    },
}
