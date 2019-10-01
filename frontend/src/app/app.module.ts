import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { NgModule } from '@angular/core';
import { HttpClientModule } from '@angular/common/http';

import { AppRoutingModule } from './app-routing.module';
import { AppComponent } from './app.component';

import { ApiModule, BASE_PATH, Configuration } from './backend';

export function cfgFactory() {
    return new Configuration();
}

@NgModule({
    declarations: [
        AppComponent
    ],
    imports: [
        BrowserModule,
        BrowserAnimationsModule,
        HttpClientModule,
        AppRoutingModule,

        ApiModule.forRoot(cfgFactory),
    ],
    providers: [{ provide: BASE_PATH, useValue: 'http://localhost:5000/api' }],
    bootstrap: [AppComponent]
})
export class AppModule { }
