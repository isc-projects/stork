import { NgModule } from '@angular/core';
import { Routes, RouterModule } from '@angular/router';

import { AuthGuard } from './auth.guard';
import { DashboardComponent } from './dashboard/dashboard.component';
import { LoginScreenComponent } from './login-screen/login-screen.component';


const routes: Routes = [
    {
        path: '',
        component: DashboardComponent,
        pathMatch: 'full',
        canActivate: [AuthGuard],
    },
    {
        path: 'login',
        component: LoginScreenComponent
    },


    // otherwise redirect to home
    { path: '**', redirectTo: '' }
];

@NgModule({
  imports: [RouterModule.forRoot(routes)],
  exports: [RouterModule]
})
export class AppRoutingModule { }
