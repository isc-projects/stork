import { Component, OnInit, AfterViewInit, ElementRef } from '@angular/core';

import { environment } from '../../environments/environment';

import {SwaggerUIBundle} from 'swagger-ui-dist';


@Component({
  selector: 'app-swagger-ui',
  templateUrl: './swagger-ui.component.html',
  styleUrls: ['./swagger-ui.component.sass']
})
export class SwaggerUiComponent implements OnInit, AfterViewInit {

    constructor() { }

    ngOnInit() {
    }

    ngAfterViewInit() {
        const ui = SwaggerUIBundle({
            url: '/swagger.json',
            dom_id: '#swagger-container',
            deepLinking: true,
            presets: [
                SwaggerUIBundle.presets.apis
            ],
        });
    }
}
