import { NgModule } from '@angular/core'
import { Routes, RouterModule } from '@angular/router'

import { AuthGuard } from './auth.guard'
import { DashboardComponent } from './dashboard/dashboard.component'
import { LoginScreenComponent } from './login-screen/login-screen.component'
import { MachinesPageComponent } from './machines-page/machines-page.component'
import { UsersPageComponent } from './users-page/users-page.component'
import { AppsPageComponent } from './apps-page/apps-page.component'
import { ProfilePageComponent } from './profile-page/profile-page.component'
import { PasswordChangePageComponent } from './password-change-page/password-change-page.component'
import { HostsPageComponent } from './hosts-page/hosts-page.component'
import { SubnetsPageComponent } from './subnets-page/subnets-page.component'
import { SharedNetworksPageComponent } from './shared-networks-page/shared-networks-page.component'
import { SettingsPageComponent } from './settings-page/settings-page.component'
import { EventsPageComponent } from './events-page/events-page.component'
import { ForbiddenPageComponent } from './forbidden-page/forbidden-page.component'
import { LogViewPageComponent } from './log-view-page/log-view-page.component'
import { LeaseSearchPageComponent } from './lease-search-page/lease-search-page.component'
import { KeaDaemonConfigurationPageComponent } from './kea-daemon-configuration-page/kea-daemon-configuration-page.component'
import { ConfigCheckerPreferencePageComponent } from './config-checker-preference-page/config-checker-preference-page.component'
import { CommunicationStatusPageComponent } from './communication-status-page/communication-status-page.component'
import { KeaGlobalConfigurationPageComponent } from './kea-global-configuration-page/kea-global-configuration-page.component'
import { VersionPageComponent } from './version-page/version-page.component'
import { ZonesPageComponent } from './zones-page/zones-page.component'
import { ConfigMigrationPageComponent } from './config-migration-page/config-migration-page.component'
import { NotFoundPageComponent } from './not-found-page/not-found-page.component'

const routes: Routes = [
    {
        path: '',
        pathMatch: 'full',
        redirectTo: 'dashboard',
    },
    {
        path: 'dashboard',
        component: DashboardComponent,
        canActivate: [AuthGuard],
    },
    {
        path: 'login',
        component: LoginScreenComponent,
    },
    {
        path: 'logout',
        component: LoginScreenComponent,
    },
    {
        path: 'machines',
        pathMatch: 'full',
        redirectTo: 'machines/all',
    },
    {
        path: 'machines/:id',
        component: MachinesPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'machine' },
    },
    {
        path: 'communication',
        component: CommunicationStatusPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'communication' },
    },
    {
        path: 'apps',
        pathMatch: 'full',
        redirectTo: 'apps/all',
    },
    {
        path: 'apps/:id',
        component: AppsPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'app' },
    },
    {
        path: 'apps/:appId/daemons/:daemonId/config',
        component: KeaDaemonConfigurationPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'daemon-config' },
    },
    {
        path: 'dhcp/leases',
        component: LeaseSearchPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'leases' },
    },
    {
        path: 'dhcp/hosts',
        pathMatch: 'full',
        redirectTo: 'dhcp/hosts/all',
    },
    {
        path: 'dhcp/hosts/:id',
        component: HostsPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'host-reservation' },
    },
    {
        path: 'dhcp/subnets',
        pathMatch: 'full',
        redirectTo: 'dhcp/subnets/all',
    },
    {
        path: 'dhcp/subnets/:id',
        component: SubnetsPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'subnet' },
    },
    {
        path: 'dhcp/shared-networks',
        pathMatch: 'full',
        redirectTo: 'dhcp/shared-networks/all',
    },
    {
        path: 'dhcp/shared-networks/:id',
        component: SharedNetworksPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'shared-network' },
    },
    {
        path: 'dns/zones',
        pathMatch: 'full',
        redirectTo: 'dns/zones/all',
    },
    {
        path: 'dns/zones/:id',
        component: ZonesPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'zones' },
    },
    {
        path: 'apps/:appId/daemons/:daemonId/global-config',
        component: KeaGlobalConfigurationPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'daemon-global-config' },
    },
    {
        path: 'profile',
        component: ProfilePageComponent,
        canActivate: [AuthGuard],
    },
    {
        path: 'profile/settings',
        component: ProfilePageComponent,
        canActivate: [AuthGuard],
    },
    {
        path: 'profile/password',
        component: PasswordChangePageComponent,
        canActivate: [AuthGuard],
        data: { key: 'user-password' },
    },
    {
        path: 'users',
        pathMatch: 'full',
        redirectTo: 'users/list',
    },
    {
        path: 'users/:id',
        component: UsersPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'users' },
    },
    {
        path: 'settings',
        component: SettingsPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'stork-settings' },
    },
    {
        path: 'events',
        component: EventsPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'events' },
    },
    {
        path: 'forbidden',
        component: ForbiddenPageComponent,
        canActivate: [AuthGuard],
    },
    {
        path: 'logs/:id',
        component: LogViewPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'logs' },
    },
    {
        path: 'review-checkers',
        component: ConfigCheckerPreferencePageComponent,
        canActivate: [AuthGuard],
        data: { key: 'global-config-checkers' },
    },
    {
        path: 'versions',
        component: VersionPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'versions' },
    },
    {
        path: 'config-migrations',
        component: ConfigMigrationPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'migrations' },
    },
    {
        path: 'config-migrations/:id',
        component: ConfigMigrationPageComponent,
        canActivate: [AuthGuard],
        data: { key: 'migrations' },
    },
    {
        path: 'swagger-ui',
        loadChildren: () => import('./swagger-ui/swagger-ui.module').then((m) => m.SwaggerUiModule),
        canActivate: [AuthGuard],
        data: { key: 'swagger' },
    },
    {
        path: '**',
        component: NotFoundPageComponent,
        canActivate: [AuthGuard],
    },
]

@NgModule({
    imports: [RouterModule.forRoot(routes, {})],
    exports: [RouterModule],
})
export class AppRoutingModule {}
