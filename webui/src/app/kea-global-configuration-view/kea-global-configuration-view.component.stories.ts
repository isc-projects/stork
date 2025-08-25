import { moduleMetadata, Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { KeaGlobalConfigurationViewComponent } from './kea-global-configuration-view.component'
import { FieldsetModule } from 'primeng/fieldset'
import { CascadedParametersBoardComponent } from '../cascaded-parameters-board/cascaded-parameters-board.component'
import { ButtonModule } from 'primeng/button'
import { TableModule } from 'primeng/table'
import { TooltipModule } from 'primeng/tooltip'
import { DhcpOptionSetViewComponent } from '../dhcp-option-set-view/dhcp-option-set-view.component'
import { TreeModule } from 'primeng/tree'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { PopoverModule } from 'primeng/popover'
import { TagModule } from 'primeng/tag'
import { ManagedAccessDirective } from '../managed-access.directive'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { toastDecorator } from '../utils-stories'
import { ToastModule } from 'primeng/toast'
import { ParameterViewComponent } from '../parameter-view/parameter-view.component'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { provideNoopAnimations } from '@angular/platform-browser/animations'

export default {
    title: 'App/KeaGlobalConfigurationView',
    component: KeaGlobalConfigurationViewComponent,
    decorators: [
        applicationConfig({
            providers: [
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideNoopAnimations(),
                MessageService,
            ],
        }),
        moduleMetadata({
            imports: [
                ButtonModule,
                FieldsetModule,
                TableModule,
                TooltipModule,
                TreeModule,
                PopoverModule,
                TagModule,
                ManagedAccessDirective,
                ToastModule,
            ],
            declarations: [
                CascadedParametersBoardComponent,
                KeaGlobalConfigurationViewComponent,
                DhcpOptionSetViewComponent,
                HelpTipComponent,
                ParameterViewComponent,
                PlaceholderPipe,
            ],
        }),
        toastDecorator,
    ],
} as Meta

type Story = StoryObj<KeaGlobalConfigurationViewComponent>

export const KeaGlobalConfiguration: Story = {
    args: {
        dhcpParameters: [
            {
                name: 'Server1',
                parameters: [
                    {
                        cacheThreshold: 0.25,
                        cacheMaxAge: 1000,
                        clientClass: 'baz',
                        requireClientClasses: ['foo', 'bar'],
                        ddnsGeneratedPrefix: 'myhost',
                        ddnsOverrideClientUpdate: true,
                    },
                    {
                        cacheThreshold: 0.25,
                        cacheMaxAge: 1000,
                        clientClass: 'fbi',
                        requireClientClasses: ['abc'],
                        ddnsGeneratedPrefix: 'his',
                        ddnsOverrideClientUpdate: false,
                    },
                    {
                        cacheMaxAge: 1000,
                        requireClientClasses: ['abc'],
                        ddnsGeneratedPrefix: 'example',
                        ddnsOverrideClientUpdate: true,
                    },
                ],
            },
        ],
    },
}
