import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';

import { SwaggerUiRoutingModule } from './swagger-ui-routing.module';
import { SwaggerUiComponent } from './swagger-ui.component';


@NgModule({
  declarations: [
    SwaggerUiComponent
  ],
  imports: [
    CommonModule,
    SwaggerUiRoutingModule,
  ]
})
export class SwaggerUiModule { }
