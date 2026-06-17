import { applicationConfig, Meta, StoryObj } from '@storybook/angular'
import { UserFormComponent } from './user-form.component'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { MessageService } from 'primeng/api'
import { UserFormState } from '../forms/user-form'
import { Group, User } from '../backend'
import { TabType } from '../tab-view/tab-view.component'
import { expect, within } from 'storybook/test'

const groups: Group[] = [
    {
        id: 1,
        name: 'super-admin',
        description: 'super-admin group description',
    },
    {
        id: 2,
        name: 'admin',
        description: 'admin group description',
    },
    {
        id: 3,
        name: 'read-only',
        description: 'read-only group description',
    },
]

const emptyFormState = new UserFormState()

const meta: Meta<UserFormComponent> = {
    title: 'App/UserForm',
    component: UserFormComponent,
    decorators: [
        applicationConfig({
            providers: [provideHttpClient(withInterceptorsFromDi()), MessageService],
        }),
    ],
    parameters: {},
    args: {
        groups: groups,
        formState: emptyFormState,
    },
}

export default meta

type Story = StoryObj<UserFormComponent>

const user: User = {
    id: 1,
    authenticationMethodId: 'internal',
    login: 'foo',
    email: 'foo@bar.org',
    name: 'Foo',
    lastname: 'Bar',
    groups: [1],
    changePassword: true,
}

const externalUser: User = {
    id: 2,
    authenticationMethodId: 'oidc',
    login: 'foo',
    email: 'foo@bar.org',
    name: 'Foo',
    lastname: 'Bar',
    groups: [1],
}

const externalUserExternallyManagedGroups: User = {
    id: 3,
    authenticationMethodId: 'oidc',
    login: 'foo',
    email: 'foo@bar.org',
    name: 'Foo',
    lastname: 'Bar',
    groups: [1],
    externallyManagedGroups: true,
}

export const CreateInternalUser: Story = {
    args: {
        tabType: TabType.New,
    },
}

export const UpdateInternalUser: Story = {
    args: {
        tabType: TabType.Edit,
        user: user,
    },
}

export const UpdateExternalUser: Story = {
    args: {
        tabType: TabType.Edit,
        user: externalUser,
    },
    // Test that group can be edited for external user.
    play: async ({ canvasElement, userEvent }) => {
        // Arrange
        const canvas = within(canvasElement)

        // Act + Assert
        const saveBtn = await canvas.findByRole('button', { name: 'Save' })
        await expect(saveBtn).not.toBeDisabled()
        await expect(canvas.queryByText(/group assignment for this user is disabled/i)).toBeNull()
        const groupSelect = await canvas.findByRole('combobox')
        await userEvent.click(groupSelect)
        const options = await canvas.findAllByRole('option')
        await expect(options.length).toBeGreaterThan(2)
    },
}

export const UpdateExternalUserExternallyManagedGroups: Story = {
    args: {
        tabType: TabType.Edit,
        user: externalUserExternallyManagedGroups,
    },
    // Test that group can't be edited for external user with externally managed groups.
    play: async ({ canvasElement, userEvent }) => {
        // Arrange
        const canvas = within(canvasElement)

        // Act + Assert
        const saveBtn = await canvas.findByRole('button', { name: 'Save' })
        await expect(saveBtn).toBeDisabled()
        await expect(canvas.getByText(/group assignment for this user is disabled/i)).toBeInTheDocument()
        const groupSelect = await canvas.findByRole('combobox')
        await expect(userEvent.click(groupSelect)).rejects.toThrow(/unable to perform pointer interaction/i)
    },
}
