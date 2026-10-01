import { enableProdMode, importProvidersFrom, provideZoneChangeDetection } from '@angular/core'

import AuraBluePreset, { cfgFactory } from './app/app.config'
import { environment } from './environments/environment'
import { HTTP_INTERCEPTORS, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { AuthInterceptor } from './app/auth-interceptor'
import { BASE_PATH, ApiModule } from './app/backend'
import { getBaseApiPath } from './app/utils'
import { ConfirmationService, MessageService } from '@openng/optimus-ui/api'
import { provideOptimus } from '@openng/optimus-ui/config'
import { BrowserModule, bootstrapApplication } from '@angular/platform-browser'
import { provideAnimations } from '@angular/platform-browser/animations'
import { AppRoutingModule } from './app/app-routing.module'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { FontAwesomeModule } from '@fortawesome/angular-fontawesome'
import { ButtonModule } from '@openng/optimus-ui/button'
import { MenubarModule } from '@openng/optimus-ui/menubar'
import { PanelModule } from '@openng/optimus-ui/panel'
import { TableModule } from '@openng/optimus-ui/table'
import { ProgressBarModule } from '@openng/optimus-ui/progressbar'
import { DialogModule } from '@openng/optimus-ui/dialog'
import { InputTextModule } from '@openng/optimus-ui/inputtext'
import { SelectModule } from '@openng/optimus-ui/select'
import { ToastModule } from '@openng/optimus-ui/toast'
import { MessageModule } from '@openng/optimus-ui/message'
import { MenuModule } from '@openng/optimus-ui/menu'
import { ProgressSpinnerModule } from '@openng/optimus-ui/progressspinner'
import { TooltipModule } from '@openng/optimus-ui/tooltip'
import { PasswordModule } from '@openng/optimus-ui/password'
import { CardModule } from '@openng/optimus-ui/card'
import { SplitButtonModule } from '@openng/optimus-ui/splitbutton'
import { FieldsetModule } from '@openng/optimus-ui/fieldset'
import { PopoverModule } from '@openng/optimus-ui/popover'
import { ToggleSwitchModule } from '@openng/optimus-ui/toggleswitch'
import { BreadcrumbModule } from '@openng/optimus-ui/breadcrumb'
import { PaginatorModule } from '@openng/optimus-ui/paginator'
import { SelectButtonModule } from '@openng/optimus-ui/selectbutton'
import { DividerModule } from '@openng/optimus-ui/divider'
import { TagModule } from '@openng/optimus-ui/tag'
import { ToggleButtonModule } from '@openng/optimus-ui/togglebutton'
import { MultiSelectModule } from '@openng/optimus-ui/multiselect'
import { CheckboxModule } from '@openng/optimus-ui/checkbox'
import { ConfirmDialogModule } from '@openng/optimus-ui/confirmdialog'
import { TextareaModule } from '@openng/optimus-ui/textarea'
import { TreeModule } from '@openng/optimus-ui/tree'
import { ChipModule } from '@openng/optimus-ui/chip'
import { DataViewModule } from '@openng/optimus-ui/dataview'
import { ChartModule } from '@openng/optimus-ui/chart'
import { AccordionModule } from '@openng/optimus-ui/accordion'
import { TreeTableModule } from '@openng/optimus-ui/treetable'
import { BadgeModule } from '@openng/optimus-ui/badge'
import { SkeletonModule } from '@openng/optimus-ui/skeleton'
import { FloatLabelModule } from '@openng/optimus-ui/floatlabel'
import { AutoCompleteModule } from '@openng/optimus-ui/autocomplete'
import { InputNumberModule } from '@openng/optimus-ui/inputnumber'
import { AppComponent } from './app/app.component'

if (environment.production) {
    enableProdMode()
}

bootstrapApplication(AppComponent, {
    providers: [
        importProvidersFrom(
            BrowserModule,
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
            FloatLabelModule,
            AutoCompleteModule,
            InputNumberModule
        ),
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
        provideOptimus({
            theme: {
                preset: AuraBluePreset,
                options: {
                    darkModeSelector: '.dark',
                    cssLayer: {
                        name: 'optimus',
                        order: 'low, optimus, high',
                    },
                },
            },
        }),
        provideAnimations(),
        provideZoneChangeDetection(),
    ],
}).catch((err) => console.error(err))
