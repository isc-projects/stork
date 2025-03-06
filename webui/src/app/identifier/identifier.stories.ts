import { IdentifierComponent } from './identifier.component'

import { StoryObj, Meta, moduleMetadata, applicationConfig } from '@storybook/angular'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { FormsModule } from '@angular/forms'
import { ByteCharacterComponent } from '../byte-character/byte-character.component'
import { RouterTestingModule } from '@angular/router/testing'

export default {
    title: 'App/Identifier',
    component: IdentifierComponent,
    decorators: [
        applicationConfig({
            providers: [],
        }),
        moduleMetadata({
            imports: [FormsModule, ToggleButtonModule, NoopAnimationsModule, RouterTestingModule],
            declarations: [ByteCharacterComponent],
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
