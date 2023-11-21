import { moduleMetadata, Meta, Story, applicationConfig } from '@storybook/angular'
import { SettingsPageComponent } from './settings-page.component'
import { importProvidersFrom } from '@angular/core'
import { provideAnimations, provideNoopAnimations } from '@angular/platform-browser/animations'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { FieldsetModule } from 'primeng/fieldset'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { MessagesModule } from 'primeng/messages'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { RouterTestingModule } from '@angular/router/testing'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { MessageService } from 'primeng/api'
import { ButtonModule } from 'primeng/button'
import { DividerModule } from 'primeng/divider'

export default {
    title: 'App/SettingsPage',
    component: SettingsPageComponent,
    decorators: [
        applicationConfig({
            providers: [
                MessageService,
                importProvidersFrom(HttpClientTestingModule),
                provideNoopAnimations(),
                provideAnimations(),
            ],
        }),
        moduleMetadata({
            imports: [
                BreadcrumbModule,
                ButtonModule,
                DividerModule,
                FieldsetModule,
                FormsModule,
                MessagesModule,
                OverlayPanelModule,
                ReactiveFormsModule,
                RouterTestingModule,
            ],
            declarations: [BreadcrumbsComponent, HelpTipComponent, SettingsPageComponent],
        }),
    ],
} as Meta

const Template: Story<SettingsPageComponent> = (args: SettingsPageComponent) => ({
    props: args,
})

export const Settings = Template.bind({})
