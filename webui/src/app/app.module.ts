// Angular modules
import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { NgModule } from '@angular/core';
import { HttpClientModule } from '@angular/common/http';
import { FormsModule } from '@angular/forms';
import { ReactiveFormsModule } from '@angular/forms'

// Other 3rd-party modules
import { FontAwesomeModule } from '@fortawesome/angular-fontawesome';

// PrimeNG modules
import {ButtonModule} from 'primeng/button';
import {MenubarModule} from 'primeng/menubar';
import {PanelModule} from 'primeng/panel';
import {TableModule} from 'primeng/table';
import {TabViewModule} from 'primeng/tabview';
import {ProgressBarModule} from 'primeng/progressbar';
import {DialogModule} from 'primeng/dialog';
import {InputTextModule} from 'primeng/inputtext';
import {DropdownModule} from 'primeng/dropdown';
import {MessageService} from 'primeng/api';
import {ToastModule} from 'primeng/toast';

// Generated API modules
import { ApiModule, BASE_PATH, Configuration } from './backend';

// Stork modules
import { environment } from './../environments/environment';
import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';
import { LoginScreenComponent } from './login-screen/login-screen.component';
import { DashboardComponent } from './dashboard/dashboard.component';
import { HostsTableComponent } from './hosts-table/hosts-table.component';
import { SwaggerUiComponent } from './swagger-ui/swagger-ui.component';
import { MachinesPageComponent } from './machines-page/machines-page.component';
import { LocaltimePipe } from './localtime.pipe';

export function cfgFactory() {
    const params: ConfigurationParameters = {
        withCredentials: true
    }
    return new Configuration(params);
}

@NgModule({
    declarations: [
        AppComponent,
        LoginScreenComponent,
        DashboardComponent,
        HostsTableComponent,
        SwaggerUiComponent,
        MachinesPageComponent,
        LocaltimePipe
    ],
    imports: [
        BrowserModule,
        BrowserAnimationsModule,
        HttpClientModule,
        FormsModule,
        AppRoutingModule,
        FormsModule,
        ReactiveFormsModule,

        FontAwesomeModule,

        ApiModule.forRoot(cfgFactory),

        ButtonModule,
        MenubarModule,
        PanelModule,
        TableModule,
        TabViewModule,
        ProgressBarModule,
        DialogModule,
        InputTextModule,
        DropdownModule,
        ToastModule,
    ],
    providers: [{ provide: BASE_PATH, useValue: environment.apiUrl }, MessageService],
    bootstrap: [AppComponent]
})
export class AppModule { }
