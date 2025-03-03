import { ComponentFixture, TestBed } from '@angular/core/testing'

import { ConfigMigrationPageComponent } from './config-migration-page.component'
import { RouterTestingModule } from '@angular/router/testing'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { TabMenuModule } from 'primeng/tabmenu'
import { ConfigMigrationTableComponent } from '../config-migration-table/config-migration-table.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { ButtonModule } from 'primeng/button'
import { TableModule } from 'primeng/table'
import { PluralizePipe } from '../pipes/pluralize.pipe'

describe('ConfigMigrationPageComponent', () => {
    let component: ConfigMigrationPageComponent
    let fixture: ComponentFixture<ConfigMigrationPageComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [MessageService],
            imports: [
                RouterTestingModule,
                HttpClientTestingModule,
                BreadcrumbModule,
                TabMenuModule,
                OverlayPanelModule,
                ButtonModule,
                TableModule,
            ],
            declarations: [
                ConfigMigrationPageComponent,
                BreadcrumbsComponent,
                ConfigMigrationTableComponent,
                HelpTipComponent,
                PluralizePipe,
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(ConfigMigrationPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
