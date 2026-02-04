import { By } from '@angular/platform-browser'
import { ComponentFixture, fakeAsync, TestBed, tick, waitForAsync } from '@angular/core/testing'

import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { EventsPageComponent } from './events-page.component'
import { EventsService } from '../backend/api/events.service'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { ConfirmationService, MessageService } from 'primeng/api'
import { ServerSentEventsService, ServerSentEventsTestingService } from '../server-sent-events.service'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideRouter, Router } from '@angular/router'

describe('EventsPageComponent', () => {
    let component: EventsPageComponent
    let fixture: ComponentFixture<EventsPageComponent>
    let router: Router

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                ConfirmationService,
                EventsService,
                MessageService,
                { provide: ServerSentEventsService, useClass: ServerSentEventsTestingService },
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideRouter([{ path: 'events', component: EventsPageComponent }]),
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(EventsPageComponent)
        router = fixture.debugElement.injector.get(Router)
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

    it('should retrieve filter parameters from route', fakeAsync(() => {
        router.navigate(['events'], { queryParams: { level: '2', machineId: '3', daemonName: 'dhcp4', userId: '4' } })
        tick()
        component.ngOnInit()

        expect(component.machineId).toBe(3)
        expect(component.daemonName).toBe('dhcp4')
        expect(component.userId).toBe(4)
    }))
})
