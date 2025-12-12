import { componentWrapperDecorator } from '@storybook/angular'
import { MessageService, ToastMessageOptions } from 'primeng/api'

/**
 * Wraps the component with the PrimeNG toast handler.
 * The module metadata decorator of the story must import ToastModule.
 */
export const toastDecorator = componentWrapperDecorator(
    (story) => `<div>
        <p-toast></p-toast>
        <div>${story}</div>
    </div>`
)

export class MessageServiceMock extends MessageService {
    /**
     * Overrides PrimeNG MessageService "add" behavior.
     * @param message message to be displayed
     */
    add(message: ToastMessageOptions) {
        if (
            message.detail.includes('URL parameter id not supported') ||
            message.detail.includes('URL parameter viewMode not supported') ||
            message.detail.includes('URL parameter args not supported') ||
            message.detail.includes('URL parameter globals not supported')
        ) {
            // Do not display toast messages about not supported URL parameters that are internal Storybook params.
            return
        }

        super.add(message)
    }
}
