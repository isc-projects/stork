import { ComponentFixture, TestBed } from '@angular/core/testing'

import { ConfigMigrationPageComponent } from './config-migration-page.component'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { RouterTestingModule } from '@angular/router/testing'
import { MessageService } from 'primeng/api'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { TabMenuModule } from 'primeng/tabmenu'
import { ConfigMigrationTableComponent } from '../config-migration-table/config-migration-table.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'

describe('ConfigMigrationPageComponent', () => {
    let component: ConfigMigrationPageComponent
    let fixture: ComponentFixture<ConfigMigrationPageComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [
                ConfigMigrationPageComponent,
                BreadcrumbsComponent,
                ConfigMigrationTableComponent,
                HelpTipComponent,
            ],
            imports: [RouterTestingModule, TabMenuModule, BreadcrumbModule, OverlayPanelModule],
            providers: [provideHttpClient(withInterceptorsFromDi()), provideHttpClientTesting(), MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(ConfigMigrationPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
