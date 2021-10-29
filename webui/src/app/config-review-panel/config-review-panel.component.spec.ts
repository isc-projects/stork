import { fakeAsync, tick, ComponentFixture, TestBed } from '@angular/core/testing'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { of, throwError } from 'rxjs'
import { DividerModule } from 'primeng/divider'
import { PaginatorModule } from 'primeng/paginator'
import { TagModule } from 'primeng/tag'
import { MessageService } from 'primeng/api'
import { ConfigReviewPanelComponent } from './config-review-panel.component'
import { EventTextComponent } from '../event-text/event-text.component'
import { ServicesService } from '../backend'

describe('ConfigReviewPanelComponent', () => {
    let component: ConfigReviewPanelComponent
    let fixture: ComponentFixture<ConfigReviewPanelComponent>
    let servicesApi: ServicesService
    let msgService: MessageService

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [ServicesService, MessageService],
            imports: [DividerModule, HttpClientTestingModule, NoopAnimationsModule, PaginatorModule, TagModule],
            declarations: [ConfigReviewPanelComponent, EventTextComponent],
        }).compileComponents()
    })

    beforeEach(() => {
        fixture = TestBed.createComponent(ConfigReviewPanelComponent)
        component = fixture.componentInstance
        servicesApi = fixture.debugElement.injector.get(ServicesService)
        msgService = fixture.debugElement.injector.get(MessageService)
        component.daemonId = 0
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should handle a communication error', fakeAsync(() => {
        // Simulate an error returned by the server and ensure that the
        // error message is displayed over the message service.
        spyOn(servicesApi, 'getDaemonConfigReports').and.returnValue(throwError({ status: 404 }))
        spyOn(msgService, 'add')

        // This call should trigger the API call.
        component.ngOnInit()
        tick()

        // Make sure that the API call was triggered and that appropriate
        // state was set within the error handler.
        expect(servicesApi.getDaemonConfigReports).toHaveBeenCalled()
        expect(msgService.add).toHaveBeenCalled()
        expect(component.start).toBe(0)
        expect(component.limit).toBe(5)
        expect(component.total).toBe(0)
        expect(component.reports.length).toBe(0)

        // Ensure that the report list box is not displayed.
        const configReportsDiv = fixture.debugElement.query(By.css('#config-reports-div'))
        expect(configReportsDiv).toBeFalsy()

        // A message indicating that no reports were found should be shown.
        const emptyListText = fixture.debugElement.query(By.css('#empty-list-text'))
        expect(emptyListText.properties.innerText).toBe('No issues found for this daemon.')
    }))

    it('should display empty list message when null reports returned', fakeAsync(() => {
        // The API call returns a null items value when no reports were found.
        const fakeReports: any = {
            items: null,
            total: 0,
        }
        spyOn(servicesApi, 'getDaemonConfigReports').and.returnValue(of(fakeReports))

        // Try to get the reports.
        component.ngOnInit()
        tick()

        // The reports and the total values should be set accordingly.
        expect(component.reports).toBeNull()
        expect(component.total).toBe(0)

        // Make sure that the view was properly updated.
        fixture.detectChanges()

        const configReportsDiv = fixture.debugElement.query(By.css('#config-reports-div'))
        expect(configReportsDiv).toBeFalsy()

        const emptyListText = fixture.debugElement.query(By.css('#empty-list-text'))
        expect(emptyListText.properties.innerText).toBe('No issues found for this daemon.')
    }))

    it('should display empty list message when empty reports returned', fakeAsync(() => {
        // Let's return an empty list of reports and make sure that
        // the information about no returned reports is displayed.
        const fakeReports: any = {
            items: [],
            total: 0,
        }
        spyOn(servicesApi, 'getDaemonConfigReports').and.returnValue(of(fakeReports))

        // Try to get the reports.
        component.ngOnInit()
        tick()

        // Make sure that the state was properly updated.
        expect(component.reports).toBeTruthy()
        expect(component.reports.length).toBe(0)
        expect(component.total).toBe(0)

        // Refresh the view.
        fixture.detectChanges()

        const configReportsDiv = fixture.debugElement.query(By.css('#config-reports-div'))
        expect(configReportsDiv).toBeFalsy()

        const emptyListText = fixture.debugElement.query(By.css('#empty-list-text'))
        expect(emptyListText.properties.innerText).toBe('No issues found for this daemon.')
    }))

    it('should get config reports', fakeAsync(() => {
        // Generate and return several reports.
        const fakeReports: any = {
            items: new Array(),
            total: 5,
        }
        for (let i = 0; i < 5; i++) {
            const report = {
                checker: 'checker_no_' + i,
                content: 'test content no ' + i,
            }
            fakeReports.items.push(report)
        }
        spyOn(servicesApi, 'getDaemonConfigReports').and.returnValue(of(fakeReports))
        spyOn(component.updateTotal, 'emit')

        component.daemonId = 1

        // Try to get the reports.
        component.ngOnInit()
        tick()

        // Make sure the reports were recorded.
        expect(component.reports).toBeTruthy()
        expect(component.reports.length).toBe(5)
        expect(component.total).toBe(5)

        // Make sure that the event notifying about the new total number of
        // reports was emitted.
        expect(component.updateTotal.emit).toHaveBeenCalledWith({ daemonId: 1, total: 5 })

        // Refresh the view.
        fixture.detectChanges()

        // The box holding the report list should be visible.
        const configReportsDiv = fixture.debugElement.query(By.css('#config-reports-div'))
        expect(configReportsDiv).toBeTruthy()

        // It should contain 5 badges with checker names.
        const checkerTags = configReportsDiv.queryAll(By.css('p-tag'))
        expect(checkerTags.length).toBe(5)

        // It should contain 5 config reports.
        const reportContents = configReportsDiv.queryAll(By.css('app-event-text'))
        expect(reportContents.length).toBe(5)

        // Validate the checker names and the report contents.
        for (let i = 0; i < 5; i++) {
            expect(checkerTags[i].nativeElement.innerText).toBe(fakeReports.items[i].checker)
            expect(reportContents[i].nativeElement.innerText).toBe(fakeReports.items[i].content)
        }
    }))

    it('should get daemon configs after on pagination', fakeAsync(() => {
        spyOn(servicesApi, 'getDaemonConfigReports').and.callThrough()

        // Set the input for the paginate function.
        component.daemonId = 123
        const event = { first: 2, rows: 5 }
        component.paginate(event)
        tick()

        // This function should execute getDaemonConfigReports with appropriate
        // parameters.
        expect(servicesApi.getDaemonConfigReports).toHaveBeenCalledWith(component.daemonId, event.first, event.rows)
    }))
})
