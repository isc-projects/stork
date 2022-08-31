import { fakeAsync, tick, ComponentFixture, TestBed } from '@angular/core/testing'
import { HttpStatusCode } from '@angular/common/http'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { of, throwError } from 'rxjs'
import { DividerModule } from 'primeng/divider'
import { PaginatorModule } from 'primeng/paginator'
import { TagModule } from 'primeng/tag'
import { PanelModule } from 'primeng/panel'
import { ButtonModule } from 'primeng/button'
import { MessageService } from 'primeng/api'
import { ConfigReviewPanelComponent } from './config-review-panel.component'
import { EventTextComponent } from '../event-text/event-text.component'
import { ServicesService } from '../backend'
import { LocaltimePipe } from '../localtime.pipe'
import { DialogModule } from 'primeng/dialog'
import { TableModule } from 'primeng/table'
import { ChipModule } from 'primeng/chip'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { ConfigCheckerPreferenceUpdaterComponent } from '../config-checker-preference-updater/config-checker-preference-updater.component'
import { ConfigCheckerPreferencePickerComponent } from '../config-checker-preference-picker/config-checker-preference-picker.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'

describe('ConfigReviewPanelComponent', () => {
    let component: ConfigReviewPanelComponent
    let fixture: ComponentFixture<ConfigReviewPanelComponent>
    let servicesApi: ServicesService
    let msgService: MessageService

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [ServicesService, MessageService],
            imports: [
                ButtonModule,
                DividerModule,
                HttpClientTestingModule,
                NoopAnimationsModule,
                PaginatorModule,
                PanelModule,
                TagModule,
                TableModule,
                ChipModule,
                OverlayPanelModule,
            ],
            declarations: [
                ConfigReviewPanelComponent,
                ConfigCheckerPreferenceUpdaterComponent,
                ConfigCheckerPreferencePickerComponent,
                EventTextComponent,
                LocaltimePipe,
                HelpTipComponent,
            ],
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

    it('should handle a communication error while initially fetching the reports', fakeAsync(() => {
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
        expect(component.refreshFailed).toBeTruthy()

        fixture.detectChanges()

        // A message indicating that we were unable to fetch the reports should
        // be displayed.
        const statusText = fixture.debugElement.query(By.css('#status-text'))
        expect(statusText.properties.innerText).toBe(
            'An error occurred while fetching the configuration review reports.'
        )
    }))

    it('should display empty list message when null reports are returned', fakeAsync(() => {
        // The API call returns a null items value when no reports were found.
        const fakeReports: any = {
            status: HttpStatusCode.Ok,
            body: {
                items: null,
                total: 0,
                review: {
                    createdAt: '2021-11-18',
                },
            },
        }
        spyOn(servicesApi, 'getDaemonConfigReports').and.returnValue(of(fakeReports))

        // Try to get the reports.
        component.ngOnInit()
        tick()

        // The reports, review information and the total values should be set accordingly.
        expect(component.reports).toBeNull()
        expect(component.total).toBe(0)
        expect(component.review).toBeTruthy()

        // Make sure that the view was properly updated.
        fixture.detectChanges()

        // Ensure that the review button is present.
        const reviewButton = fixture.debugElement.query(By.css('#review-button'))
        expect(reviewButton).toBeTruthy()

        const statusText = fixture.debugElement.query(By.css('#status-text'))
        expect(statusText.properties.innerText).toBe('No configuration issues found for this daemon.')
    }))

    it('should display empty list message when empty reports list is returned', fakeAsync(() => {
        // Let's return an empty list of reports and make sure that
        // the information about no returned reports is displayed.
        const fakeReports: any = {
            status: HttpStatusCode.Ok,
            body: {
                items: [],
                total: 0,
                review: {
                    createdAt: '2021-11-18',
                },
            },
        }
        spyOn(servicesApi, 'getDaemonConfigReports').and.returnValue(of(fakeReports))

        // Try to get the reports.
        component.ngOnInit()
        tick()

        // Make sure that the state was properly updated.
        expect(component.reports).toBeTruthy()
        expect(component.reports.length).toBe(0)
        expect(component.total).toBe(0)
        expect(component.review).toBeTruthy()

        // Refresh the view.
        fixture.detectChanges()

        // Ensure that the review button is present.
        const reviewButton = fixture.debugElement.query(By.css('#review-button'))
        expect(reviewButton).toBeTruthy()

        const statusText = fixture.debugElement.query(By.css('#status-text'))
        expect(statusText.properties.innerText).toBe('No configuration issues found for this daemon.')
    }))

    it('should get and display config reports', fakeAsync(() => {
        // Generate and return several reports.
        const fakeReports: any = {
            status: HttpStatusCode.Ok,
            body: {
                items: new Array(),
                total: 5,
                review: {
                    createdAt: '2021-11-18',
                },
            },
        }
        for (let i = 0; i < 5; i++) {
            const report = {
                checker: 'checker_no_' + i,
                content: 'test content no ' + i,
            }
            fakeReports.body.items.push(report)
        }
        spyOn(servicesApi, 'getDaemonConfigReports').and.returnValue(of(fakeReports))

        component.daemonId = 1

        // Try to get the reports.
        component.ngOnInit()
        tick()

        // Make sure the reports were recorded.
        expect(component.reports).toBeTruthy()
        expect(component.reports.length).toBe(5)
        expect(component.total).toBe(5)
        expect(component.review).toBeTruthy()

        // Refresh the view.
        fixture.detectChanges()

        // Ensure that the review button is present.
        const reviewButton = fixture.debugElement.query(By.css('#review-button'))
        expect(reviewButton).toBeTruthy()

        // It should contain the config review summary text.
        const reviewSummaryDiv = fixture.debugElement.query(By.css('#review-summary-div'))
        expect(reviewSummaryDiv).toBeTruthy()
        expect(reviewSummaryDiv.properties.innerText).toContain('5 reports generated at 2021-11-18')

        // It should contain 5 badges with checker names.
        const checkerTags = fixture.debugElement.queryAll(By.css('p-tag'))
        expect(checkerTags.length).toBe(5)

        // It should contain 5 config reports.
        const reportContents = fixture.debugElement.queryAll(By.css('app-event-text'))
        expect(reportContents.length).toBe(5)

        // Validate the checker names and the report contents.
        for (let i = 0; i < 5; i++) {
            expect(checkerTags[i].nativeElement.innerText).toBe(fakeReports.body.items[i].checker)
            expect(reportContents[i].nativeElement.innerText).toBe(fakeReports.body.items[i].content)
        }
    }))

    it('should get daemon configs on pagination', fakeAsync(() => {
        spyOn(servicesApi, 'getDaemonConfigReports').and.callThrough()

        // Set the input for the paginate function.
        component.daemonId = 123
        const event = { first: 2, rows: 5 }
        const observe: any = 'response'
        component.paginate(event)
        tick()

        // This function should execute getDaemonConfigReports with appropriate
        // parameters.
        expect(servicesApi.getDaemonConfigReports).toHaveBeenCalledWith(
            component.daemonId,
            event.first,
            event.rows,
            observe
        )
    }))

    it('should re-run review and refresh config reports', fakeAsync(() => {
        const putResponse: any = {
            status: HttpStatusCode.Accepted,
        }
        spyOn(servicesApi, 'putDaemonConfigReview').and.returnValue(of(putResponse))

        // Simulate the case when the was not performed for the daemon yet,
        const getResponse: any = {
            status: HttpStatusCode.NoContent,
        }
        spyOn(servicesApi, 'getDaemonConfigReports').and.returnValue(of(getResponse))

        // Run review with no delays between retries.
        component.runReview(false)
        tick()

        // Ensure that the appropriate API calls were made. The second
        // call follows the success (Accepted) response to the first
        // call.
        const observe: any = 'response'
        expect(servicesApi.putDaemonConfigReview).toHaveBeenCalledWith(component.daemonId, observe)
        expect(servicesApi.getDaemonConfigReports).toHaveBeenCalled()

        expect(component.reports).toBeTruthy()
        expect(component.reports.length).toBe(0)
        expect(component.review).toBeFalsy()
        expect(component.total).toBe(0)
        expect(component.start).toBe(0)
        expect(component.limit).toBe(5)
        expect(component.refreshFailed).toBeFalse()

        fixture.detectChanges()

        // Ensure that the review button is present.
        const reviewButton = fixture.debugElement.query(By.css('#review-button'))
        expect(reviewButton).toBeTruthy()
    }))

    it('should report an error when review request fails', fakeAsync(() => {
        // Simulate the situation that the reports were already fetched.
        component.reports = [
            {
                checker: 'checker',
                content: 'content',
            },
        ]
        component.total = 1
        component.review = {
            createdAt: '2021-11-18',
        }

        // Simulate an error returned by the server and ensure that the
        // error message is displayed over the message service.
        spyOn(servicesApi, 'putDaemonConfigReview').and.returnValue(throwError({ status: 404 }))
        spyOn(msgService, 'add')

        // This call should trigger the API call without delays between
        // the retries.
        component.runReview(false)
        tick()

        // Ensure that the API call is triggered and that the error
        // message is displayed.
        expect(servicesApi.putDaemonConfigReview).toHaveBeenCalled()
        expect(msgService.add).toHaveBeenCalled()

        // The error should not affect already presented reports.
        expect(component.reports).toBeTruthy()
        expect(component.reports.length).toBe(1)
        expect(component.total).toBe(1)

        fixture.detectChanges()

        // Ensure that the review button is present.
        const reviewButton = fixture.debugElement.query(By.css('#review-button'))
        expect(reviewButton).toBeTruthy()
    }))

    it('should warn after several refresh retries', fakeAsync(() => {
        // Simulate the case that the server returns HTTP Accepted status
        // code endlessly. It should eventually cause the client to
        // stop retrying.
        const fakeResponse: any = {
            status: HttpStatusCode.Accepted,
        }
        spyOn(servicesApi, 'getDaemonConfigReports').and.returnValues(of(fakeResponse))
        spyOn(msgService, 'add')

        // Try to get the reports.
        component.refreshDaemonConfigReports(null, false)
        tick()

        // Ensure that the API call is made and that the warning message
        // is displayed informing that the user should try refreshing
        // later.
        expect(servicesApi.getDaemonConfigReports).toHaveBeenCalled()
        expect(msgService.add).toHaveBeenCalled()

        expect(component.busy).toBeFalse()
        expect(component.refreshFailed).toBeTrue()

        fixture.detectChanges()

        // Ensure that the review button is absent.
        const reviewButton = fixture.debugElement.query(By.css('#review-button'))
        expect(reviewButton).toBeFalsy()

        // Ensure that the refresh button is present.
        const refreshButton = fixture.debugElement.query(By.css('#refresh-button'))
        expect(refreshButton).toBeTruthy()
    }))

    it('should open the config review checkers panel with minimal layout on click the button', () => {
        const buttonElement = fixture.debugElement.query(By.css('button[label=Checkers]'))
        expect(buttonElement).not.toBeNull()
        // The checker picker shouldn't exist yet.
        let pickerElement = fixture.debugElement.query(By.directive(ConfigCheckerPreferenceUpdaterComponent))
        expect(pickerElement).toBeNull()

        buttonElement.triggerEventHandler('click', null)
        fixture.detectChanges()

        // The checker picker should be presented.
        pickerElement = fixture.debugElement.query(By.directive(ConfigCheckerPreferenceUpdaterComponent))
        expect(pickerElement).not.toBeNull()
        const pickerComponent = pickerElement.componentInstance as ConfigCheckerPreferenceUpdaterComponent
        expect(pickerComponent.minimal).toBeTrue()
        expect(pickerComponent.daemonID).not.toBeNull()
    })
})
