import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { SwaggerUiComponent } from './swagger-ui.component';

const routes: Routes = [{ path: '', component: SwaggerUiComponent }];

@NgModule({
  imports: [RouterModule.forChild(routes)],
  exports: [RouterModule]
})
export class SwaggerUiRoutingModule { }
