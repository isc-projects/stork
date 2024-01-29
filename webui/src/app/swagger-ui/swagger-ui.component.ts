import { Component, AfterViewInit } from '@angular/core'

import { SwaggerUIBundle } from 'swagger-ui-dist'

@Component({
    selector: 'app-swagger-ui',
    templateUrl: './swagger-ui.component.html',
    styleUrls: ['./swagger-ui.component.sass'],
})
export class SwaggerUiComponent implements AfterViewInit {
    constructor() {}

    ngAfterViewInit() {
        SwaggerUIBundle({
            url: '/swagger.json',
            dom_id: '#swagger-container',
            deepLinking: true,
            presets: [SwaggerUIBundle.presets.apis],
        })
    }
}
