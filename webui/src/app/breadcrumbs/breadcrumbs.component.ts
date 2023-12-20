import { Component, OnInit, Input } from '@angular/core'
import { Title } from '@angular/platform-browser'
import { MenuItem } from 'primeng/api'

@Component({
    selector: 'app-breadcrumbs',
    templateUrl: './breadcrumbs.component.html',
    styleUrls: ['./breadcrumbs.component.sass'],
})
export class BreadcrumbsComponent implements OnInit {
    @Input() items: any
    home: MenuItem = {
        icon: 'pi pi-home',
        routerLink: '/',
    }

    constructor(private titleService: Title) {}

    ngOnInit(): void {
        let title = ''
        for (const item of this.items) {
            title += item.label + ' / '
        }
        title = title.slice(0, -3) + ' - Stork'
        this.titleService.setTitle(title)
    }
}
