import { componentWrapperDecorator } from '@storybook/angular'

/**
 * Wraps the component with the PrimeNG toast handler.
 * The module metadata decorator of the story must to import ToastModule.
 */
export const toastDecorator = componentWrapperDecorator(
    (story) => `<div>
        <p-toast></p-toast>
        <div>${story}</div>
    </div>`
)
