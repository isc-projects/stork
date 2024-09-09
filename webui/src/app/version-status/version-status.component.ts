import { Component, Input } from '@angular/core'

@Component({
    selector: 'app-version-status',
    templateUrl: './version-status.component.html',
    styleUrl: './version-status.component.sass',
})
export class VersionStatusComponent {
    @Input({ required: true }) app: 'kea' | 'bind' | 'stork'

    @Input({ required: true }) version: string
}
