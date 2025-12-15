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
     * Overrides PrimeNG MessageService "add" method.
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

/**
 * Type representing typical Stork REST API response with paged data.
 */
export type EntitiesResponse = { items: any[]; total: number }

/**
 * Mocks entities filtering by matching text that is usually done on backend side.
 * @param response response with the entities to be filtered
 * @param request storybook-addon-mock request object containing searchParams
 * @param matchingField name of the field inside entity item object that will be matched against searchParam text
 * @return entities object response with items filtered by text
 */
export function mockedFilterByText(response: EntitiesResponse, request: any, matchingField: string): EntitiesResponse {
    if (request.searchParams?.text && response.items?.length) {
        const filteredItems = response.items.filter((item) => {
            return (<string>item[matchingField] ?? '').includes(request.searchParams.text)
        })
        return { items: filteredItems, total: filteredItems.length }
    }

    return response
}
