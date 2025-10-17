import { MachinesPageComponent } from './machines-page.component'
import { applicationConfig, Meta, moduleMetadata, StoryObj } from '@storybook/angular'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { ConfirmationService, MessageService } from 'primeng/api'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { DialogModule } from 'primeng/dialog'
import { ButtonModule } from 'primeng/button'
import { TabViewComponent } from '../tab-view/tab-view.component'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { provideRouter } from '@angular/router'
import { MachinesTableComponent } from '../machines-table/machines-table.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { TableModule } from 'primeng/table'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { PanelModule } from 'primeng/panel'
import { IconFieldModule } from 'primeng/iconfield'
import { InputIconModule } from 'primeng/inputicon'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { TriStateCheckboxComponent } from '../tri-state-checkbox/tri-state-checkbox.component'
import { ManagedAccessDirective } from '../managed-access.directive'
import { PopoverModule } from 'primeng/popover'
import { SelectButtonModule } from 'primeng/selectbutton'
import { MenuModule } from 'primeng/menu'
import { InputTextModule } from 'primeng/inputtext'
import { BadgeModule } from 'primeng/badge'
import { TagModule } from 'primeng/tag'
import { FormsModule } from '@angular/forms'
import { AuthService } from '../auth.service'

const meta: Meta<MachinesPageComponent> = {
    component: MachinesPageComponent,
    decorators: [
        applicationConfig({
            providers: [
                provideHttpClient(withInterceptorsFromDi()),
                MessageService,
                ConfirmationService,
                provideRouter([
                    {
                        path: 'machines',
                        pathMatch: 'full',
                        redirectTo: 'machines/all',
                    },
                    {
                        path: 'machines/:id',
                        component: MachinesPageComponent,
                    },
                    {
                        path: 'iframe.html',
                        component: MachinesPageComponent,
                    },
                ]),
                { provide: AuthService, useValue: { hasPrivilege: () => true } },
            ],
        }),
        moduleMetadata({
            declarations: [BreadcrumbsComponent, MachinesTableComponent, PluralizePipe, HelpTipComponent],
            imports: [
                DialogModule,
                ButtonModule,
                TabViewComponent,
                ConfirmDialogModule,
                BreadcrumbModule,
                TableModule,
                PanelModule,
                IconFieldModule,
                InputIconModule,
                TriStateCheckboxComponent,
                ManagedAccessDirective,
                PopoverModule,
                SelectButtonModule,
                MenuModule,
                InputTextModule,
                BadgeModule,
                TagModule,
                FormsModule,
            ],
        }),
    ],
}

export default meta
type Story = StoryObj<MachinesPageComponent>

export const EmptyList: Story = {
    parameters: {
        mockData: [
            {
                url: 'http://localhost/api/settings',
                method: 'GET',
                status: 200,
                response: () => ({ enableMachineRegistration: true }),
            },
            {
                url: 'http://localhost/api/machines?start=:start&limit=:limit',
                method: 'GET',
                status: 200,
                response: () => ({ items: [] }),
            },
            {
                url: 'http://localhost/api/machines?start=:start&limit=:limit&text=:text',
                method: 'GET',
                status: 200,
                response: () => ({ items: [] }),
            },
            {
                url: 'http://localhost/api/machines?start=:start&limit=:limit&text=:text&authorized=:authorized',
                method: 'GET',
                status: 200,
                response: () => ({ items: [] }),
            },
            {
                url: 'http://localhost/api/machines?start=:start&limit=:limit&authorized=:authorized',
                method: 'GET',
                status: 200,
                response: () => ({ items: [] }),
            },
            {
                url: 'http://localhost/api/machines/unauthorized/count',
                method: 'GET',
                status: 200,
                response: () => 0,
            },
            {
                url: 'http://localhost/api/machines-server-token',
                method: 'GET',
                status: 200,
                response: () => ({token: "randomMachineToken"}),
            },
            {
                url: 'http://localhost/api/machines-server-token',
                method: 'PUT',
                status: 200,
                response: () => ({token: "regeneratedRandomMachineToken"}),
            },
        ],
    },
}
