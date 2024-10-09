import { ComponentFixture, TestBed } from '@angular/core/testing'

import { VersionPageComponent } from './version-page.component'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { PanelModule } from 'primeng/panel'
import { TableModule } from 'primeng/table'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { ButtonModule } from 'primeng/button'
import { RouterModule } from '@angular/router'

describe('VersionPageComponent', () => {
    let component: VersionPageComponent
    let fixture: ComponentFixture<VersionPageComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [
                HttpClientTestingModule,
                PanelModule,
                TableModule,
                BreadcrumbModule,
                OverlayPanelModule,
                BrowserAnimationsModule,
                ButtonModule,
                RouterModule.forRoot([
                    {
                        path: 'versions',
                        component: VersionPageComponent,
                    },
                ]),
            ],
            declarations: [VersionPageComponent, BreadcrumbsComponent, HelpTipComponent],
            providers: [MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(VersionPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
