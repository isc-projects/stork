import { Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { KeaGlobalConfigurationViewComponent } from './kea-global-configuration-view.component'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { MessageService } from 'primeng/api'
import { toastDecorator } from '../utils-stories'
import { provideAnimations } from '@angular/platform-browser/animations'

export default {
    title: 'App/KeaGlobalConfigurationView',
    component: KeaGlobalConfigurationViewComponent,
    decorators: [
        applicationConfig({
            providers: [provideHttpClient(withInterceptorsFromDi()), provideAnimations(), MessageService],
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
