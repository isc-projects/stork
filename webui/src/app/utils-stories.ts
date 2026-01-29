import { componentWrapperDecorator } from '@storybook/angular'
import { AuthService } from './auth.service'

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

/**
 * Type representing typical Stork REST API response with paged data.
 */
export type EntitiesResponse<T> = { items: T[]; total: number }

export interface EntitiesRequest {
    searchParams?: { text: string }
}

/**
 * Mocks entities filtering by matching text that is usually done on backend side.
 * @param response response with the entities to be filtered
 * @param request storybook-addon-mock request object containing searchParams
 * @param matchingField name of the field inside entity item object that will be matched against searchParam text
 * @return entities object response with items filtered by text
 */
export function mockedFilterByText<T>(response: EntitiesResponse<T>, request: EntitiesRequest, matchingField: keyof T): EntitiesResponse<T> {
    if (request.searchParams?.text && response.items?.length) {
        const filteredItems = response.items.filter((item) => {
            return (<string>item[matchingField] ?? '').includes(request.searchParams.text)
        })
        return { items: filteredItems, total: filteredItems.length }
    }

    return response
}

/**
 * Mocks AuthService by extending its normal implementation,
 * with the exception of superAdmin, isAdmin and isInReadOnlyGroup methods.
 * Overridden methods will return values kept in global variables.
 * Global variables values are controlled by authServiceDecorator.
 */
export class MockedAuthService extends AuthService {
    /**
     * Static variables keeping state of selected user role.
     */
    static isSuperAdmin = true
    static isAdmin = false
    static isReadOnly = false

    /**
     * Returns whether current user belongs to super-admin group.
     */
    superAdmin() {
        return MockedAuthService.isSuperAdmin
    }

    /**
     * Returns whether current user belongs to admin group.
     */
    isAdmin() {
        return MockedAuthService.isAdmin
    }

    /**
     * Returns whether current user belongs to read-only group.
     */
    isInReadOnlyGroup() {
        return MockedAuthService.isReadOnly
    }
}

/**
 * Wraps the component so that isSuperAdmin, isReadOnly and isAdmin global variables
 * are controlled by Storybook "globals.role" setting.
 */
export const authServiceDecorator = componentWrapperDecorator(
    (story) => story,
    ({ globals }) => {
        if (!globals.role) {
            MockedAuthService.isSuperAdmin = true
            MockedAuthService.isReadOnly = false
            MockedAuthService.isAdmin = false
            return
        }

        switch (globals.role) {
            case 'super-admin':
                MockedAuthService.isSuperAdmin = true
                MockedAuthService.isAdmin = false
                MockedAuthService.isReadOnly = false
                break
            case 'admin':
                MockedAuthService.isSuperAdmin = false
                MockedAuthService.isAdmin = true
                MockedAuthService.isReadOnly = false
                break
            case 'read-only':
                MockedAuthService.isReadOnly = true
                MockedAuthService.isSuperAdmin = false
                MockedAuthService.isAdmin = false
                break
            default:
                MockedAuthService.isSuperAdmin = true
                MockedAuthService.isReadOnly = false
                MockedAuthService.isAdmin = false
                break
        }
    }
)
