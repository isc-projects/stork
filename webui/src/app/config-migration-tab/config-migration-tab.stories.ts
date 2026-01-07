import { Meta, StoryObj, applicationConfig } from '@storybook/angular'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { toastDecorator } from '../utils-stories'
import { ConfigMigrationTabComponent } from './config-migration-tab.component'
import { MessageService } from 'primeng/api'
import { provideRouter, withHashLocation } from '@angular/router'

export default {
    title: 'App/ConfigMigrationTab',
    component: ConfigMigrationTabComponent,
    decorators: [
        applicationConfig({
            providers: [
                provideHttpClient(withInterceptorsFromDi()),
                MessageService,
                provideRouter([{ path: '**', component: ConfigMigrationTabComponent }], withHashLocation()),
            ],
        }),
        toastDecorator,
    ],
} as Meta

type Story = StoryObj<ConfigMigrationTabComponent>

export const RunningMigration: Story = {
    args: {
        migration: {
            id: 123,
            startDate: new Date().toISOString(),
            endDate: null,
            canceling: false,
            processedItemsCount: 35,
            totalItemsCount: 100,
            errors: {
                total: 0,
                items: [],
            },
            elapsedTime: '10m30s',
            estimatedLeftTime: '19m45s',
            authorId: 42,
            authorLogin: 'admin',
        },
    },
}

export const CancelingMigration: Story = {
    args: {
        migration: {
            id: 456,
            startDate: new Date().toISOString(),
            endDate: null,
            canceling: true,
            processedItemsCount: 67,
            totalItemsCount: 100,
            errors: {
                total: 0,
                items: [],
            },
            elapsedTime: '15m22s',
            estimatedLeftTime: '7m12s',
            authorId: 42,
            authorLogin: 'admin',
        },
    },
}

export const CompletedMigration: Story = {
    args: {
        migration: {
            id: 789,
            startDate: new Date(Date.now() - 1800000).toISOString(), // 30 minutes ago
            endDate: new Date().toISOString(),
            canceling: false,
            processedItemsCount: 100,
            totalItemsCount: 100,
            errors: {
                total: 0,
                items: [],
            },
            elapsedTime: '30m0s',
            estimatedLeftTime: '0s',
            authorId: 42,
            authorLogin: 'admin',
        },
    },
}

export const FailedMigration: Story = {
    args: {
        migration: {
            id: 24,
            startDate: new Date(Date.now() - 600000).toISOString(), // 10 minutes ago
            endDate: new Date().toISOString(),
            canceling: false,
            processedItemsCount: 45,
            totalItemsCount: 100,
            generalError: 'Migration failed due to connectivity issues',
            errors: {
                total: 3,
                items: [
                    { id: 1, error: 'Failed to process host: timeout', label: 'host-1', causeEntity: 'host' },
                    { id: 2, error: 'Failed to process host: invalid data', label: 'host-2', causeEntity: 'host' },
                    {
                        id: 3,
                        error: 'Failed to process host: connection refused',
                        label: 'host-3',
                        causeEntity: 'host',
                    },
                ],
            },
            elapsedTime: '10m0s',
            estimatedLeftTime: '0s',
            authorId: 42,
            authorLogin: 'admin',
        },
    },
}

export const FailedMigrationWithManyErrors: Story = {
    args: {
        migration: {
            id: 42,
            startDate: new Date(Date.now() - 900000).toISOString(), // 15 minutes ago
            endDate: new Date().toISOString(),
            canceling: false,
            processedItemsCount: 78,
            totalItemsCount: 100,
            generalError: 'Migration terminated due to exceeding error threshold',
            errors: {
                total: 10,
                items: Array(10)
                    .fill(0)
                    .map((_, i) => ({
                        hostId: i + 1,
                        error: `Error code ${1000 + i}: Host configuration validation failed`,
                        label: `host-${i + 1}`,
                        type: 'host',
                    })),
            },
            elapsedTime: '15m0s',
            estimatedLeftTime: '0s',
            authorId: 42,
            authorLogin: 'admin',
        },
    },
}
