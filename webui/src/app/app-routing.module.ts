import { NgModule } from '@angular/core'
import { Routes, RouterModule, RouteReuseStrategy, ActivatedRouteSnapshot, DetachedRouteHandle } from '@angular/router'

import { AuthGuard } from './auth.guard'
import { DashboardComponent } from './dashboard/dashboard.component'
import { LoginScreenComponent } from './login-screen/login-screen.component'
import { SwaggerUiComponent } from './swagger-ui/swagger-ui.component'
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
        path: 'swagger-ui',
        component: SwaggerUiComponent,
        canActivate: [AuthGuard],
        data: { key: 'swagger' },
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

export class CustomRouteReuseStrategy implements RouteReuseStrategy {
    /**
     * Array of specific components that will have custom route reuse strategy.
     *
     * @private
     */
    private specificComponents: any[] = [
        HostsPageComponent,
        SubnetsPageComponent,
        SharedNetworksPageComponent,
        MachinesPageComponent,
    ]

    /**
     * The point of this CustomRouteReuseStrategy is to skip route reuse in specific cases.
     * Hence, this method is not implemented. Nothing will be retrieved.
     */
    retrieve(): DetachedRouteHandle | null {
        return null
    }

    /**
     * The point of this CustomRouteReuseStrategy is to skip route reuse in specific cases.
     * Hence, this method always returns false.
     */
    shouldAttach(): boolean {
        return false
    }

    /**
     * The point of this CustomRouteReuseStrategy is to skip route reuse in specific cases.
     * Hence, this method always returns false.
     */
    shouldDetach(): boolean {
        return false
    }

    /**
     * Determines whether the route should be reused.
     * It returns false when navigation happens between two same specific components,
     * e.g. between two HostsPageComponents,
     * and when either of below conditions apply:
     *   - curr and future routes contain param 'id=all' (for specific components it means that list view tab
     *     with index 0 is displayed).
     *   - future route queryParamMap contains 'gs' key i.e. global search was used
     *     (e.g. future route looks like dhcp/hosts/all?text=foobar&gs=true).
     * For other routes, true is returned whenever current route and future
     * route have exactly the same routeConfig. In this case, default Angular
     * route reuse strategy will work as usual.
     * @param future route to which we are trying to navigate
     * @param curr route from which we are leaving
     */
    shouldReuseRoute(future: ActivatedRouteSnapshot, curr: ActivatedRouteSnapshot): boolean {
        if (
            this.specificComponents.includes(future.component) &&
            future.component === curr.component &&
            (future.queryParamMap.has('gs') ||
                (curr.paramMap.get('id')?.includes('all') && future.paramMap.get('id')?.includes('all')))
        ) {
            // Do not reuse route when navigation happens between two same specific components,
            // (e.g. between two HostsPageComponents)
            // and when either of below conditions apply:
            //   - curr and future routes display list view tab (tab index 0)
            //   - future route queryParamMap contains 'gs' key i.e. global search was used
            //     (e.g. future route looks like dhcp/hosts/all?text=foobar&gs=true).
            return false
        }
        return future.routeConfig === curr.routeConfig
    }

    /**
     * The point of this CustomRouteReuseStrategy is to skip route reuse in specific cases.
     * Hence, this method is not implemented. Nothing will be stored.
     */
    store(): void {
        // no-op
    }
}
