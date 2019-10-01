import { Component } from '@angular/core';

import { DefaultService } from './backend/api/default.service';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.sass']
})
export class AppComponent {
    title = 'Stork';

    version = 'not yet';

    constructor(protected api: DefaultService) {
    }

    ngOnInit() {
        this.api.versionGet().subscribe(data => {
            console.info(data);
            this.version = data.version;
        });
    }
}
