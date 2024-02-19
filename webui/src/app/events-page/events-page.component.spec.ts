import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { EventsPageComponent } from './events-page.component'
import { EventsService } from '../backend/api/events.service'
import { EventsPanelComponent } from '../events-panel/events-panel.component'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { FormsModule } from '@angular/forms'
import { SelectButtonModule } from 'primeng/selectbutton'
import { DropdownModule } from 'primeng/dropdown'
import { TableModule } from 'primeng/table'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { RouterTestingModule } from '@angular/router/testing'
import { ServerSentEventsService, ServerSentEventsTestingService } from '../server-sent-events.service'

describe('EventsPageComponent', () => {
    let component: EventsPageComponent
    let fixture: ComponentFixture<EventsPageComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                EventsService,
                MessageService,
                { provide: ServerSentEventsService, useClass: ServerSentEventsTestingService },
            ],
            declarations: [
                BreadcrumbsComponent,
                EventsPageComponent,
                EventsPageComponent,
                EventsPanelComponent,
                HelpTipComponent,
            ],
            imports: [
                HttpClientTestingModule,
                FormsModule,
                SelectButtonModule,
                DropdownModule,
                TableModule,
                BreadcrumbModule,
                OverlayPanelModule,
                RouterTestingModule,
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(EventsPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('Monitoring')
        expect(breadcrumbsComponent.items[1].label).toEqual('Events')
    })
})
