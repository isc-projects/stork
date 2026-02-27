import { Component, OnInit, Input, inject } from '@angular/core'
import { Title } from '@angular/platform-browser'
import { MenuItem } from 'primeng/api'
import { Breadcrumb } from 'primeng/breadcrumb'
import { HelpTipComponent } from '../help-tip/help-tip.component'

@Component({
    selector: 'app-breadcrumbs',
    templateUrl: './breadcrumbs.component.html',
    styleUrls: ['./breadcrumbs.component.sass'],
    imports: [Breadcrumb, HelpTipComponent],
})
export class BreadcrumbsComponent implements OnInit {
    private titleService = inject(Title)

    @Input() items: any
    home: MenuItem = {
        icon: 'pi pi-home',
        routerLink: '/',
    }

    /**
     * Custom breadcrumbs style.
     */
    breadcrumbDesign = {
        root: {
            itemColor: '{content.color}',
        },
    }

    ngOnInit(): void {
        let title = ''
        for (const item of this.items) {
            title += item.label + ' / '
        }
        title = title.slice(0, -3) + ' - Stork'
        this.titleService.setTitle(title)
    }
}
