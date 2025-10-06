// Angular modules
import { BrowserModule } from '@angular/platform-browser'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { NgModule } from '@angular/core'
import { HTTP_INTERCEPTORS, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'

// Other 3rd-party modules
import { FontAwesomeModule } from '@fortawesome/angular-fontawesome'

// PrimeNG modules
import { ButtonModule } from 'primeng/button'
import { MenubarModule } from 'primeng/menubar'
import { PanelModule } from 'primeng/panel'
import { TableModule } from 'primeng/table'
import { ProgressBarModule } from 'primeng/progressbar'
import { DialogModule } from 'primeng/dialog'
import { InputTextModule } from 'primeng/inputtext'
import { SelectModule } from 'primeng/select'
import { ConfirmationService, MessageService } from 'primeng/api'
import { ToastModule } from 'primeng/toast'
import { MessageModule } from 'primeng/message'
import { MenuModule } from 'primeng/menu'
import { ChipModule } from 'primeng/chip'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { TooltipModule } from 'primeng/tooltip'
import { PasswordModule } from 'primeng/password'
import { CardModule } from 'primeng/card'
import { SplitButtonModule } from 'primeng/splitbutton'
import { FieldsetModule } from 'primeng/fieldset'
import { PopoverModule } from 'primeng/popover'
import { ToggleSwitchModule } from 'primeng/toggleswitch'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { PaginatorModule } from 'primeng/paginator'
import { SelectButtonModule } from 'primeng/selectbutton'
import { DividerModule } from 'primeng/divider'
import { TagModule } from 'primeng/tag'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { MultiSelectModule } from 'primeng/multiselect'
import { CheckboxModule } from 'primeng/checkbox'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { TextareaModule } from 'primeng/textarea'
import { TreeModule } from 'primeng/tree'
import { DataViewModule } from 'primeng/dataview'
import { ChartModule } from 'primeng/chart'
import { AccordionModule } from 'primeng/accordion'
import { TreeTableModule } from 'primeng/treetable'
import { SkeletonModule } from 'primeng/skeleton'
import { FloatLabelModule } from 'primeng/floatlabel'
import { AutoCompleteModule } from 'primeng/autocomplete'
import { InputNumberModule } from 'primeng/inputnumber'
import Aura from '@primeng/themes/aura'
import { definePreset } from '@primeng/themes'
import { providePrimeNG } from 'primeng/config'
import { IconField } from 'primeng/iconfield'
import { InputIcon } from 'primeng/inputicon'

// Generated API modules
import { ApiModule, BASE_PATH, Configuration, ConfigurationParameters } from './backend'

// Stork modules
import { environment } from './../environments/environment'
import { getBaseApiPath } from './utils'
import { AppRoutingModule } from './app-routing.module'
import { AppComponent } from './app.component'
import { AuthInterceptor } from './auth-interceptor'
import { LoginScreenComponent } from './login-screen/login-screen.component'
import { DashboardComponent } from './dashboard/dashboard.component'
import { HostsTableComponent } from './hosts-table/hosts-table.component'
import { MachinesPageComponent } from './machines-page/machines-page.component'
import { UsersPageComponent } from './users-page/users-page.component'
import { AppsPageComponent } from './apps-page/apps-page.component'
import { AppTabComponent } from './app-tab/app-tab.component'
import { KeaAppTabComponent } from './kea-app-tab/kea-app-tab.component'
import { PasswordChangePageComponent } from './password-change-page/password-change-page.component'
import { ProfilePageComponent } from './profile-page/profile-page.component'
import { SettingsMenuComponent } from './settings-menu/settings-menu.component'
import { HaStatusComponent } from './ha-status/ha-status.component'
import { SubnetsPageComponent } from './subnets-page/subnets-page.component'
import { SharedNetworksPageComponent } from './shared-networks-page/shared-networks-page.component'
import { SubnetBarComponent } from './subnet-bar/subnet-bar.component'
import { HostsPageComponent } from './hosts-page/hosts-page.component'
import { SettingsPageComponent } from './settings-page/settings-page.component'
import { HelpTipComponent } from './help-tip/help-tip.component'
import { GlobalSearchComponent } from './global-search/global-search.component'
import { EventTextComponent } from './event-text/event-text.component'
import { EntityLinkComponent } from './entity-link/entity-link.component'
import { EventsPanelComponent } from './events-panel/events-panel.component'
import { ForbiddenPageComponent } from './forbidden-page/forbidden-page.component'
import { LogViewPageComponent } from './log-view-page/log-view-page.component'
import { AppDaemonsStatusComponent } from './app-daemons-status/app-daemons-status.component'
import { BreadcrumbsComponent } from './breadcrumbs/breadcrumbs.component'
import { EventsPageComponent } from './events-page/events-page.component'
import { RenameAppDialogComponent } from './rename-app-dialog/rename-app-dialog.component'
import { LeaseSearchPageComponent } from './lease-search-page/lease-search-page.component'
import { JsonTreeComponent } from './json-tree/json-tree.component'
import { JsonTreeRootComponent } from './json-tree-root/json-tree-root.component'
import { KeaDaemonConfigurationPageComponent } from './kea-daemon-configuration-page/kea-daemon-configuration-page.component'
import { HostTabComponent } from './host-tab/host-tab.component'
import { ConfigReviewPanelComponent } from './config-review-panel/config-review-panel.component'
import { IdentifierComponent } from './identifier/identifier.component'
import { AppOverviewComponent } from './app-overview/app-overview.component'
import { HostFormComponent } from './host-form/host-form.component'
import { DhcpOptionFormComponent } from './dhcp-option-form/dhcp-option-form.component'
import { DhcpOptionSetFormComponent } from './dhcp-option-set-form/dhcp-option-set-form.component'
import { ConfigCheckerPreferenceUpdaterComponent } from './config-checker-preference-updater/config-checker-preference-updater.component'
import { ConfigCheckerPreferencePickerComponent } from './config-checker-preference-picker/config-checker-preference-picker.component'
import { ConfigCheckerPreferencePageComponent } from './config-checker-preference-page/config-checker-preference-page.component'
import { DhcpOptionSetViewComponent } from './dhcp-option-set-view/dhcp-option-set-view.component'
import { DhcpClientClassSetFormComponent } from './dhcp-client-class-set-form/dhcp-client-class-set-form.component'
import { DhcpClientClassSetViewComponent } from './dhcp-client-class-set-view/dhcp-client-class-set-view.component'
import { DelegatedPrefixBarComponent } from './delegated-prefix-bar/delegated-prefix-bar.component'
import { HumanCountComponent } from './human-count/human-count.component'
import { LocalNumberPipe } from './pipes/local-number.pipe'
import { HumanCountPipe } from './pipes/human-count.pipe'
import { LocaltimePipe } from './pipes/localtime.pipe'
import { SurroundPipe } from './pipes/surround.pipe'
import { AccessPointKeyComponent } from './access-point-key/access-point-key.component'
import { PlaceholderPipe } from './pipes/placeholder.pipe'
import { PluralizePipe } from './pipes/pluralize.pipe'
import { SubnetTabComponent } from './subnet-tab/subnet-tab.component'
import { AddressPoolBarComponent } from './address-pool-bar/address-pool-bar.component'
import { UtilizationStatsChartComponent } from './utilization-stats-chart/utilization-stats-chart.component'
import { UtilizationStatsChartsComponent } from './utilization-stats-charts/utilization-stats-charts.component'
import { CascadedParametersBoardComponent } from './cascaded-parameters-board/cascaded-parameters-board.component'
import { SharedNetworkTabComponent } from './shared-network-tab/shared-network-tab.component'
import { HostDataSourceLabelComponent } from './host-data-source-label/host-data-source-label.component'
import { SharedParametersFormComponent } from './shared-parameters-form/shared-parameters-form.component'
import { SubnetFormComponent } from './subnet-form/subnet-form.component'
import { AddressPoolFormComponent } from './address-pool-form/address-pool-form.component'
import { PrefixPoolFormComponent } from './prefix-pool-form/prefix-pool-form.component'
import { ArrayValueSetFormComponent } from './array-value-set-form/array-value-set-form.component'
import { PriorityErrorsPanelComponent } from './priority-errors-panel/priority-errors-panel.component'
import { SharedNetworkFormComponent } from './shared-network-form/shared-network-form.component'
import { CommunicationStatusTreeComponent } from './communication-status-tree/communication-status-tree.component'
import { CommunicationStatusPageComponent } from './communication-status-page/communication-status-page.component'
import { KeaGlobalConfigurationPageComponent } from './kea-global-configuration-page/kea-global-configuration-page.component'
import { ParameterViewComponent } from './parameter-view/parameter-view.component'
import { UncamelPipe } from './pipes/uncamel.pipe'
import { UnhyphenPipe } from './pipes/unhyphen.pipe'
import { SubnetsTableComponent } from './subnets-table/subnets-table.component'
import { SharedNetworksTableComponent } from './shared-networks-table/shared-networks-table.component'
import { KeaGlobalConfigurationViewComponent } from './kea-global-configuration-view/kea-global-configuration-view.component'
import { KeaGlobalConfigurationFormComponent } from './kea-global-configuration-form/kea-global-configuration-form.component'
import { PositivePipe } from './pipes/positive.pipe'
import { VersionStatusComponent } from './version-status/version-status.component'
import { VersionPageComponent } from './version-page/version-page.component'
import { BadgeModule } from 'primeng/badge'
import { MachinesTableComponent } from './machines-table/machines-table.component'
import { ZonesPageComponent } from './zones-page/zones-page.component'
import { ByteCharacterComponent } from './byte-character/byte-character.component'
import { ZoneViewerComponent } from './zone-viewer/zone-viewer.component'
import { ZoneViewerFeederComponent } from './zone-viewer-feeder/zone-viewer-feeder.component'
import { ConfigMigrationPageComponent } from './config-migration-page/config-migration-page.component'
import { ConfigMigrationTableComponent } from './config-migration-table/config-migration-table.component'
import { ConfigMigrationTabComponent } from './config-migration-tab/config-migration-tab.component'
import { DurationPipe } from './pipes/duration.pipe'
import { ManagedAccessDirective } from './managed-access.directive'
import { NotFoundPageComponent } from './not-found-page/not-found-page.component'
import { UtilizationBarComponent } from './utilization-bar/utilization-bar.component'
import { PoolBarsComponent } from './pool-bars/pool-bars.component'
import { UnrootPipe } from './pipes/unroot.pipe'
import { OutOfPoolBarComponent } from './out-of-pool-bar/out-of-pool-bar.component'
import { DaemonNiceNamePipe } from './pipes/daemon-name.pipe'
import { Bind9DaemonComponent } from './bind9-daemon/bind9-daemon.component'
import { PdnsDaemonComponent } from './pdns-daemon/pdns-daemon.component'
import { TabViewComponent } from './tab-view/tab-view.component'
import { TriStateCheckboxComponent } from './tri-state-checkbox/tri-state-checkbox.component'
import { UserFormComponent } from './user-form/user-form.component'
import { TextFileViewerComponent } from './text-file-viewer/text-file-viewer.component'
import { Bind9ConfigPreviewComponent } from './bind9-config-preview/bind9-config-preview.component'
import { Bind9DaemonControlsComponent } from './bind9-daemon-controls/bind9-daemon-controls.component'

/** Create the OpenAPI client configuration. */
export function cfgFactory() {
    const params: ConfigurationParameters = {
        apiKeys: {},
        withCredentials: true,
    }
    return new Configuration(params)
}

const AuraBluePreset = definePreset(Aura, {
    semantic: {
        primary: {
            50: '{blue.50}',
            100: '{blue.100}',
            200: '{blue.200}',
            300: '{blue.300}',
            400: '{blue.400}',
            500: '{blue.500}',
            600: '{blue.600}',
            700: '{blue.700}',
            800: '{blue.800}',
            900: '{blue.900}',
            950: '{blue.950}',
        },
        colorScheme: {
            // Adding custom 'inverted' color scheme which mimics old PrimeNG 'surface' color scheme,
            // which for light scheme was changing from white to dark colors, and for
            // dark scheme it was changing from dark colors to white.
            // In new PrimeNG (v18 and following), the 'surface' color scheme behaves similarly for both light and dark mode,
            // i.e. it changes from white to darker colors.
            dark: {
                inverted: {
                    0: '{zinc.900}',
                    50: '{zinc.800}',
                    100: '{zinc.700}',
                    200: '{zinc.600}',
                    300: '{zinc.500}',
                    400: '{zinc.400}',
                    500: '{zinc.300}',
                    600: '{zinc.200}',
                    700: '{zinc.100}',
                    800: '{zinc.50}',
                    900: '#ffffff',
                    950: '#ffffff',
                },
            },
            light: {
                inverted: {
                    0: '#ffffff',
                    50: '{slate.50}',
                    100: '{slate.100}',
                    200: '{slate.200}',
                    300: '{slate.300}',
                    400: '{slate.400}',
                    500: '{slate.500}',
                    600: '{slate.600}',
                    700: '{slate.700}',
                    800: '{slate.800}',
                    900: '{slate.900}',
                    950: '{slate.950}',
                },
            },
        },
    },
    components: {
        // Apply primary background color for Chips instead of the default greyish surface color.
        chip: {
            colorScheme: {
                light: {
                    root: {
                        background: '{primary.100}',
                    },
                },
                dark: {
                    root: {
                        background: '{primary.400}',
                    },
                },
            },
        },
        // Make messages text lighter (500 by default).
        message: {
            colorScheme: {
                light: {
                    root: {
                        textFontWeight: '400',
                    },
                },
                dark: {
                    root: {
                        textFontWeight: '400',
                    },
                },
            },
        },
        // Apply regular padding for all panel headers.
        panel: {
            colorScheme: {
                light: {
                    root: {
                        toggleableHeaderPadding: '1.125rem',
                    },
                },
                dark: {
                    root: {
                        toggleableHeaderPadding: '1.125rem',
                    },
                },
            },
        },
        // Customize accordion header background colors and apply smaller padding for accordion panel content.
        accordion: {
            colorScheme: {
                light: {
                    root: {
                        headerBackground: '{surface.50}',
                        headerActiveBackground: '{surface.50}',
                        headerHoverBackground: '{surface.100}',
                        headerActiveHoverBackground: '{surface.100}',
                        contentPadding: '0 0.5rem 0.5rem 0.5rem',
                    },
                },
                dark: {
                    root: {
                        headerBackground: '{surface.950}',
                        headerActiveBackground: '{surface.950}',
                        headerHoverBackground: '{surface.800}',
                        headerActiveHoverBackground: '{surface.800}',
                        contentPadding: '0 0.5rem 0.5rem 0.5rem',
                    },
                },
            },
        },
    },
})

@NgModule({
    declarations: [
        AppComponent,
        LoginScreenComponent,
        DashboardComponent,
        HostsTableComponent,
        MachinesPageComponent,
        LocaltimePipe,
        UsersPageComponent,
        AppsPageComponent,
        AppTabComponent,
        KeaAppTabComponent,
        PasswordChangePageComponent,
        ProfilePageComponent,
        SettingsMenuComponent,
        HaStatusComponent,
        SubnetsPageComponent,
        SharedNetworksPageComponent,
        SubnetBarComponent,
        HostsPageComponent,
        SettingsPageComponent,
        HelpTipComponent,
        GlobalSearchComponent,
        EventTextComponent,
        EntityLinkComponent,
        EventsPanelComponent,
        ForbiddenPageComponent,
        LogViewPageComponent,
        AppDaemonsStatusComponent,
        BreadcrumbsComponent,
        EventsPageComponent,
        RenameAppDialogComponent,
        LeaseSearchPageComponent,
        JsonTreeComponent,
        JsonTreeRootComponent,
        KeaDaemonConfigurationPageComponent,
        HostTabComponent,
        ConfigReviewPanelComponent,
        IdentifierComponent,
        AppOverviewComponent,
        HostFormComponent,
        DhcpOptionFormComponent,
        DhcpOptionSetFormComponent,
        DhcpOptionSetViewComponent,
        ConfigCheckerPreferencePickerComponent,
        ConfigCheckerPreferenceUpdaterComponent,
        ConfigCheckerPreferencePageComponent,
        DhcpClientClassSetFormComponent,
        DhcpClientClassSetViewComponent,
        DelegatedPrefixBarComponent,
        HumanCountPipe,
        HumanCountComponent,
        LocalNumberPipe,
        SurroundPipe,
        AccessPointKeyComponent,
        PlaceholderPipe,
        SubnetTabComponent,
        AddressPoolBarComponent,
        UtilizationStatsChartComponent,
        UtilizationStatsChartsComponent,
        CascadedParametersBoardComponent,
        SharedNetworkTabComponent,
        SharedParametersFormComponent,
        SubnetFormComponent,
        AddressPoolFormComponent,
        PrefixPoolFormComponent,
        ArrayValueSetFormComponent,
        HostDataSourceLabelComponent,
        PriorityErrorsPanelComponent,
        PluralizePipe,
        SharedNetworkFormComponent,
        CommunicationStatusTreeComponent,
        CommunicationStatusPageComponent,
        KeaGlobalConfigurationPageComponent,
        ParameterViewComponent,
        UncamelPipe,
        UnhyphenPipe,
        SubnetsTableComponent,
        SharedNetworksTableComponent,
        KeaGlobalConfigurationViewComponent,
        KeaGlobalConfigurationFormComponent,
        PositivePipe,
        VersionStatusComponent,
        VersionPageComponent,
        MachinesTableComponent,
        ZonesPageComponent,
        ByteCharacterComponent,
        ZoneViewerComponent,
        ZoneViewerFeederComponent,
        ConfigMigrationPageComponent,
        ConfigMigrationTableComponent,
        ConfigMigrationTabComponent,
        DurationPipe,
        NotFoundPageComponent,
        UtilizationBarComponent,
        PoolBarsComponent,
        UnrootPipe,
        OutOfPoolBarComponent,
        DaemonNiceNamePipe,
        Bind9DaemonComponent,
        PdnsDaemonComponent,
    ],
    bootstrap: [AppComponent],
    imports: [
        BrowserModule,
        BrowserAnimationsModule,
        AppRoutingModule,
        FormsModule,
        ReactiveFormsModule,
        FontAwesomeModule,
        ApiModule.forRoot(cfgFactory),
        ButtonModule,
        MenubarModule,
        PanelModule,
        TableModule,
        ProgressBarModule,
        DialogModule,
        InputTextModule,
        SelectModule,
        ToastModule,
        MessageModule,
        MenuModule,
        ProgressSpinnerModule,
        TooltipModule,
        PasswordModule,
        CardModule,
        SplitButtonModule,
        FieldsetModule,
        PopoverModule,
        ToggleSwitchModule,
        BreadcrumbModule,
        PaginatorModule,
        SelectButtonModule,
        DividerModule,
        TagModule,
        ToggleButtonModule,
        MultiSelectModule,
        CheckboxModule,
        ConfirmDialogModule,
        TextareaModule,
        TreeModule,
        ChipModule,
        DataViewModule,
        ToggleButtonModule,
        ChartModule,
        AccordionModule,
        TreeTableModule,
        BadgeModule,
        SkeletonModule,
        ManagedAccessDirective,
        FloatLabelModule,
        AutoCompleteModule,
        InputNumberModule,
        TabViewComponent,
        IconField,
        InputIcon,
        TriStateCheckboxComponent,
        UserFormComponent,
        TextFileViewerComponent,
        Bind9ConfigPreviewComponent,
        Bind9DaemonControlsComponent,
    ],
    providers: [
        {
            provide: HTTP_INTERCEPTORS,
            useClass: AuthInterceptor,
            multi: true,
        },
        {
            provide: BASE_PATH,
            useValue: getBaseApiPath(environment.apiUrl),
        },
        ConfirmationService,
        MessageService,
        provideHttpClient(withInterceptorsFromDi()),
        providePrimeNG({
            theme: {
                preset: AuraBluePreset,
                options: {
                    darkModeSelector: '.dark',
                    cssLayer: {
                        name: 'primeng',
                        order: 'low, primeng, high',
                    },
                },
            },
        }),
    ],
})
export class AppModule {}

export default AuraBluePreset
