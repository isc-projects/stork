import { ByteCharacterComponent } from './byte-character.component'

import { StoryObj, Meta } from '@storybook/angular'

export default {
    title: 'App/ByteCharacter',
    component: ByteCharacterComponent,
} as Meta

type Story = StoryObj<ByteCharacterComponent>

export const PrintableLetter: Story = {
    args: {
        byteValue: 65,
    },
}

export const PrintableSymbol: Story = {
    args: {
        byteValue: 64,
    },
}

export const NonPrintable: Story = {
    args: {
        byteValue: 0,
    },
}

export const NonByteNegative: Story = {
    args: {
        byteValue: -1,
    },
}

export const NonBytePositive: Story = {
    args: {
        byteValue: 256,
    },
}

export const NaN: Story = {
    args: {
        byteValue: Number.NaN,
    },
}
