import { enableProdMode, importProvidersFrom } from '@angular/core'

import AuraBluePreset, { cfgFactory } from './app/app.config'
import { environment } from './environments/environment'
import { HTTP_INTERCEPTORS, provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { AuthInterceptor } from './app/auth-interceptor'
import { BASE_PATH, ApiModule } from './app/backend'
import { getBaseApiPath } from './app/utils'
import { ConfirmationService, MessageService } from 'primeng/api'
import { providePrimeNG } from 'primeng/config'
import { BrowserModule, bootstrapApplication } from '@angular/platform-browser'
import { provideAnimations } from '@angular/platform-browser/animations'
import { AppRoutingModule } from './app/app-routing.module'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { FontAwesomeModule } from '@fortawesome/angular-fontawesome'
import { ButtonModule } from 'primeng/button'
import { MenubarModule } from 'primeng/menubar'
import { PanelModule } from 'primeng/panel'
import { TableModule } from 'primeng/table'
import { ProgressBarModule } from 'primeng/progressbar'
import { DialogModule } from 'primeng/dialog'
import { InputTextModule } from 'primeng/inputtext'
import { SelectModule } from 'primeng/select'
import { ToastModule } from 'primeng/toast'
import { MessageModule } from 'primeng/message'
import { MenuModule } from 'primeng/menu'
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
import { ChipModule } from 'primeng/chip'
import { DataViewModule } from 'primeng/dataview'
import { ChartModule } from 'primeng/chart'
import { AccordionModule } from 'primeng/accordion'
import { TreeTableModule } from 'primeng/treetable'
import { BadgeModule } from 'primeng/badge'
import { SkeletonModule } from 'primeng/skeleton'
import { FloatLabelModule } from 'primeng/floatlabel'
import { AutoCompleteModule } from 'primeng/autocomplete'
import { InputNumberModule } from 'primeng/inputnumber'
import { IconField } from 'primeng/iconfield'
import { InputIcon } from 'primeng/inputicon'
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
            InputNumberModule,
            IconField,
            InputIcon
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
        provideAnimations(),
    ],
}).catch((err) => console.error(err))
