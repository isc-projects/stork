import { Component, OnInit } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';

import {ButtonModule} from 'primeng/button';

import { DefaultService } from '../backend/api/default.service';

@Component({
  selector: 'login-screen',
  templateUrl: './login-screen.component.html',
  styleUrls: ['./login-screen.component.sass']
})
export class LoginScreenComponent implements OnInit {

    version = 'not available';
    returnUrl: string;

    constructor(protected api: DefaultService, private route: ActivatedRoute, private router: Router) {
    }

    ngOnInit() {
        this.api.versionGet().subscribe(data => {
            console.info(data);
            this.version = data.version;
        });

        this.returnUrl = this.route.snapshot.queryParams['returnUrl'] || '/';
    }

    signIn() {
        this.router.navigate([this.returnUrl]);
    }
}
