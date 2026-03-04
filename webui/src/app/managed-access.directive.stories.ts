import { applicationConfig, Meta, moduleMetadata, StoryObj } from '@storybook/angular'
import { ManagedAccessDirective } from './managed-access.directive'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { MessageService } from 'primeng/api'
import { toastDecorator } from './utils-stories'
import { Button } from 'primeng/button'
import { TriStateCheckboxComponent } from './tri-state-checkbox/tri-state-checkbox.component'
import { ToggleSwitch } from 'primeng/toggleswitch'
import { userEvent, within, expect, waitFor } from 'storybook/test'

const meta: Meta<ManagedAccessDirective> = {
    title: 'App/ManagedAccessDirective',
    component: ManagedAccessDirective,
    render: (args) => ({
        props: args,
        template: `
<div class="flex flex-column gap-2">
    <p-button
        label="Clear Finished Migrations"
        icon="pi pi-trash"
        severity="warn"
        appAccessEntity="migrations"
        appAccessType="delete"
        (onClick)="btn.value = true"
    ></p-button>
    <div>Button clicked: <input #btn aria-label="btn-clicked-value" disabled class="w-full" [value]="false" /></div>
    <div>
        <label for="tri-state">Checkbox</label>
        <app-tri-state-checkbox
            inputID="tri-state"
            appAccessEntity="global-config-checkers"
            appAccessType="update"
            (valueChange)="checkbox.value = $event"
        ></app-tri-state-checkbox>
    </div>
    <div>Checkbox value: <input #checkbox aria-label="checkbox-value" disabled class="w-full" [value]="null" /></div>
    <div>
        <label for="monitored-switch">Toggle</label>
        <p-toggleswitch
            (onChange)="toggle.value = $event.checked"
            inputId="monitored-switch"
            appAccessEntity="daemon-monitoring"
            appAccessType="update"
        ></p-toggleswitch>
    </div>

    <div>Toggle switch value: <input #toggle aria-label="toggle-value" disabled class="w-full" [value]="false" /></div>
    <span appAccessType="update" appAccessEntity="subnet" [appHideIfNoAccess]="true"
        >This should be hidden in case of no access.</span
    >
    <span appAccessType="delete" appAccessEntity="access-point-key">You need super-admin role to see this text.</span>
</div>`,
    }),
    decorators: [
        applicationConfig({
            providers: [provideHttpClient(withInterceptorsFromDi()), MessageService],
        }),
        toastDecorator,
        moduleMetadata({
            imports: [Button, TriStateCheckboxComponent, ToggleSwitch],
        }),
    ],
}

export default meta
type Story = StoryObj<ManagedAccessDirective>

export const DifferentComponents: Story = {}

export const TestSuperAdminPrivileges: Story = {
    globals: {
        role: 'super-admin',
    },
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)

        const button = await canvas.findByRole('button')
        const btnClickedValue = await canvas.findByLabelText('btn-clicked-value')
        await expect(btnClickedValue).toHaveValue('false')

        const checkbox = await canvas.findByRole('checkbox')
        const checkboxValue = await canvas.findByLabelText('checkbox-value')
        await expect(checkboxValue).toHaveValue('')

        const toggleSwitch = await canvas.findByRole('switch')
        const toggleValue = await canvas.findByLabelText('toggle-value')
        await expect(toggleValue).toHaveValue('false')

        // Configure delay between consecutive user events to be more human-like and to give more time for PrimeNG animations when automatically testing.
        const user = userEvent.setup({ delay: 50 })

        // Act
        user.click(button)
        user.click(checkbox)
        user.click(toggleSwitch)

        // Assert
        await waitFor(() => expect(btnClickedValue).toHaveValue('true'))
        await waitFor(() => expect(checkboxValue).toHaveValue('true'))
        await waitFor(() => expect(toggleValue).toHaveValue('true'))
        await waitFor(() => expect(canvas.getByText('This should be hidden in case of no access.')))
        await waitFor(() => expect(canvas.getByText('You need super-admin role to see this text.')))
        await waitFor(() =>
            expect(
                canvas.queryByText("You don't have delete privileges to display this UI component.")
            ).not.toBeInTheDocument()
        )
    },
}

export const TestAdminPrivileges: Story = {
    globals: {
        role: 'admin',
    },
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)

        const button = await canvas.findByRole('button')
        const btnClickedValue = await canvas.findByLabelText('btn-clicked-value')
        await expect(btnClickedValue).toHaveValue('false')

        const checkbox = await canvas.findByRole('checkbox')
        const checkboxValue = await canvas.findByLabelText('checkbox-value')
        await expect(checkboxValue).toHaveValue('')

        const toggleSwitch = await canvas.findByRole('switch')
        const toggleValue = await canvas.findByLabelText('toggle-value')
        await expect(toggleValue).toHaveValue('false')

        // Configure delay between consecutive user events to be more human-like and to give more time for PrimeNG animations when automatically testing.
        const user = userEvent.setup({ delay: 50 })

        // Act
        user.click(button)
        user.click(checkbox)
        user.click(toggleSwitch)

        // Assert
        await waitFor(() => expect(btnClickedValue).toHaveValue('true'))
        await waitFor(() => expect(checkboxValue).toHaveValue('true'))
        await waitFor(() => expect(toggleValue).toHaveValue('true'))
        await waitFor(() => expect(canvas.getByText('This should be hidden in case of no access.')))
        await waitFor(() =>
            expect(canvas.queryByText('You need super-admin role to see this text.')).not.toBeInTheDocument()
        )
        await waitFor(() =>
            expect(
                canvas.queryByText("You don't have delete privileges to display this UI component.")
            ).toBeInTheDocument()
        )
    },
}

export const TestReadOnlyPrivileges: Story = {
    globals: {
        role: 'read-only',
    },
    play: async ({ canvasElement }) => {
        // Arrange
        const canvas = within(canvasElement)

        const button = await canvas.findByRole('button')
        const btnClickedValue = await canvas.findByLabelText('btn-clicked-value')
        await expect(btnClickedValue).toHaveValue('false')

        const checkbox = await canvas.findByRole('checkbox')
        const checkboxValue = await canvas.findByLabelText('checkbox-value')
        await expect(checkboxValue).toHaveValue('')

        const toggleSwitch = await canvas.findByRole('switch')
        const toggleValue = await canvas.findByLabelText('toggle-value')
        await expect(toggleValue).toHaveValue('false')

        // Configure delay between consecutive user events to be more human-like and to give more time for PrimeNG animations when automatically testing.
        const user = userEvent.setup({ delay: 50 })

        // Act
        user.click(button)
        user.click(checkbox)
        user.click(toggleSwitch)

        // Assert
        await waitFor(() => expect(btnClickedValue).toHaveValue('false'))
        await waitFor(() => expect(checkboxValue).toHaveValue(''))
        await waitFor(() => expect(toggleValue).toHaveValue('false'))
        await waitFor(() =>
            expect(canvas.queryByText('This should be hidden in case of no access.')).not.toBeInTheDocument()
        )
        await waitFor(() =>
            expect(canvas.queryByText('You need super-admin role to see this text.')).not.toBeInTheDocument()
        )
        await waitFor(() =>
            expect(
                canvas.queryByText("You don't have delete privileges to display this UI component.")
            ).toBeInTheDocument()
        )
    },
}
