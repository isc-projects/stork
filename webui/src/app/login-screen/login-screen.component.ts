import { Component, OnInit } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { HttpResponse } from "@angular/common/http";

import {ButtonModule} from 'primeng/button';

import { GeneralService } from '../backend/api/api';

@Component({
  selector: 'app-login-screen',
  templateUrl: './login-screen.component.html',
  styleUrls: ['./login-screen.component.sass']
})
export class LoginScreenComponent implements OnInit {

    version = 'not available';
    returnUrl: string;

    constructor(protected api: GeneralService, private route: ActivatedRoute, private router: Router) {
    }

    ngOnInit() {
        this.api.getVersion().subscribe(data => {
            console.info(data);
            this.version = data.version;
        });

        this.returnUrl = this.route.snapshot.queryParams.returnUrl || '/';
    }

    signIn() {
        this.api.userLoginGet("xyz", "xyz", "response").subscribe(resp => { 
        });
        this.router.navigate([this.returnUrl]);
    }
}
