import { IdentifierComponent } from './identifier.component'

import { Story, Meta, moduleMetadata, applicationConfig } from '@storybook/angular'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { FormsModule } from '@angular/forms'

export default {
    title: 'App/Identifier',
    component: IdentifierComponent,
    decorators: [
        applicationConfig({
            providers: [],
        }),
        moduleMetadata({
            imports: [FormsModule, ToggleButtonModule, NoopAnimationsModule],
        }),
    ],
} as Meta

const Template: Story<IdentifierComponent> = (args: IdentifierComponent) => ({
    props: args,
})

export const Primary = Template.bind({})

Primary.args = {
    hexValue: '73:30:6d:45:56:61:4c:75:65',
    label: 'flex-id',
}
