import { fakeAsync, tick, ComponentFixture, TestBed } from '@angular/core/testing'
import { HttpResponse, HttpStatusCode } from '@angular/common/http'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { of, throwError } from 'rxjs'
import { DividerModule } from 'primeng/divider'
import { TagModule } from 'primeng/tag'
import { ButtonModule } from 'primeng/button'
import { MessageService } from 'primeng/api'
import { ConfigReviewPanelComponent } from './config-review-panel.component'
import { EventTextComponent } from '../event-text/event-text.component'
import { ConfigReports, ServicesService } from '../backend'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { TableModule } from 'primeng/table'
import { ChipModule } from 'primeng/chip'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { ConfigCheckerPreferenceUpdaterComponent } from '../config-checker-preference-updater/config-checker-preference-updater.component'
import { ConfigCheckerPreferencePickerComponent } from '../config-checker-preference-picker/config-checker-preference-picker.component'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { DataViewModule } from 'primeng/dataview'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { FormsModule } from '@angular/forms'

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
                TagModule,
                TableModule,
                ChipModule,
                FormsModule,
                OverlayPanelModule,
                ToggleButtonModule,
                DataViewModule,
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
        component.loading = true
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
        component.loading = false
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
        expect(component.loading).toBeFalse()

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
                // Total and items are omitted because they are zero and empty.
                review: {
                    createdAt: '2021-11-18',
                },
            },
        }
        spyOn(servicesApi, 'getDaemonConfigReports').and.returnValue(of(fakeReports))

        // Try to get the reports.
        component.loading = false
        component.ngOnInit()
        tick()

        // The reports, review information and the total values should be set accordingly.
        expect(component.reports).toEqual([])
        expect(component.total).toBe(0)
        expect(component.review).toBeTruthy()

        // Make sure that the view was properly updated.
        fixture.detectChanges()

        // Ensure that the review button is present.
        const reviewButton = fixture.debugElement.query(By.css('#review-button'))
        expect(reviewButton).toBeTruthy()

        const statusText = fixture.debugElement.query(By.css('#status-text'))
        expect(statusText.properties.innerText).toBe('No configuration issues were found for this daemon.')
    }))

    it('should display empty list message when empty reports list is returned', fakeAsync(() => {
        // Let's return an empty list of reports and make sure that
        // the information about no returned reports is displayed.
        const fakeReports: any = {
            status: HttpStatusCode.Ok,
            body: {
                items: [],
                total: 0,
                totalIssues: 0,
                totalReports: 0,
                review: {
                    createdAt: '2021-11-18',
                },
            },
        }
        spyOn(servicesApi, 'getDaemonConfigReports').and.returnValue(of(fakeReports))

        // Try to get the reports.
        component.loading = false
        component.ngOnInit()
        tick()

        // Make sure that the state was properly updated.
        expect(component.reports).toBeTruthy()
        expect(component.reports.length).toBe(0)
        expect(component.total).toBe(0)
        expect(component.totalIssues).toBe(0)
        expect(component.totalReports).toBe(0)
        expect(component.review).toBeTruthy()

        // Refresh the view.
        fixture.detectChanges()

        // Ensure that the review button is present.
        const reviewButton = fixture.debugElement.query(By.css('#review-button'))
        expect(reviewButton).toBeTruthy()

        const statusText = fixture.debugElement.query(By.css('#status-text'))
        expect(statusText.properties.innerText).toBe('No configuration issues were found for this daemon.')
    }))

    it('should get and display config reports', fakeAsync(() => {
        // Generate and return several reports.
        const fakeReports: any = {
            status: HttpStatusCode.Ok,
            body: {
                items: [],
                total: 10,
                totalReports: 20,
                totalIssues: 10,
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
        component.loading = false
        component.ngOnInit()
        tick()

        // Make sure the reports were recorded.
        expect(component.reports).toBeTruthy()
        expect(component.reports.length).toBe(5)
        expect(component.total).toBe(10)
        expect(component.review).toBeTruthy()
        expect(component.totalIssues).toBe(10)
        expect(component.totalReports).toBe(20)

        // Refresh the view.
        fixture.detectChanges()

        // Ensure that the review button is present.
        const reviewButton = fixture.debugElement.query(By.css('#review-button'))
        expect(reviewButton).toBeTruthy()

        // It should contain the config review summary text.
        const reviewSummaryDiv = fixture.debugElement.query(By.css('#review-summary-div'))
        expect(reviewSummaryDiv).toBeTruthy()
        expect(reviewSummaryDiv.properties.innerText).toContain('10 issues found in 20 reports at 2021-11-18')

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

    it('should display zero counts properly', fakeAsync(() => {
        const fakeReports: any = {
            status: HttpStatusCode.Ok,
            body: {
                items: [],
                total: 5,
                totalReports: 5,
                // Total issues is omitted because it is zero.
                review: {
                    createdAt: '2021-11-18',
                },
            },
        }

        for (let i = 0; i < 5; i++) {
            const report = {
                checker: 'checker_no_' + i,
            }
            fakeReports.body.items.push(report)
        }

        spyOn(servicesApi, 'getDaemonConfigReports').and.returnValue(of(fakeReports))

        component.daemonId = 1

        // Try to get the reports.
        component.loading = false
        component.ngOnInit()
        tick()

        // Refresh the view.
        fixture.detectChanges()

        // Make sure the reports were recorded.
        expect(component.reports).toBeTruthy()
        expect(component.reports.length).toBe(5)
        expect(component.total).toBe(5)
        expect(component.review).toBeTruthy()
        expect(component.totalIssues).toBe(0)
        expect(component.totalReports).toBe(5)

        // It should contain the config review summary text.
        const reviewSummaryDiv = fixture.debugElement.query(By.css('#review-summary-div'))
        expect(reviewSummaryDiv).toBeTruthy()
        expect(reviewSummaryDiv.properties.innerText).toContain('0 issues found in 5 reports at 2021-11-18')
    }))

    it('should get daemon configs on pagination', fakeAsync(() => {
        spyOn(servicesApi, 'getDaemonConfigReports').and.returnValue(
            of(
                new HttpResponse({
                    body: {
                        total: 4,
                        totalIssues: 10,
                        totalReports: 20,
                        review: {
                            id: 2,
                            createdAt: '2022-02-21T14:09:00',
                            daemonId: 3,
                        },
                        items: [
                            {
                                checker: 'foo',
                                content: 'an issue found',
                                createdAt: '2022-02-21T14:10:00',
                                id: 4,
                            },
                            {
                                checker: 'bar',
                                createdAt: '2022-02-21T14:10:00',
                                id: 5,
                            },
                        ],
                    } as ConfigReports,
                })
            )
        )

        // Set the input for the paginate function.
        component.daemonId = 123
        component.loading = false
        const event = { first: 2, rows: 5 }
        const observe: any = 'response'
        expect(component.issuesOnly).toBeTrue()
        component.refreshDaemonConfigReports(event)
        tick()

        // This function should execute getDaemonConfigReports with appropriate
        // parameters.
        expect(servicesApi.getDaemonConfigReports).toHaveBeenCalledWith(
            component.daemonId,
            event.first,
            event.rows,
            true,
            observe
        )
        expect(component.loading).toBeFalse()
        expect(component.start).toBe(2)
        expect(component.total).toBe(4)
        expect(component.limit).toBe(5)
        expect(component.reports.length).toBe(2)
        expect(component.totalIssues).toBe(10)
        expect(component.totalReports).toBe(20)

        component.issuesOnly = false

        component.refreshDaemonConfigReports(event)
        tick()

        expect(servicesApi.getDaemonConfigReports).toHaveBeenCalledWith(
            component.daemonId,
            event.first,
            event.rows,
            false,
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
        component.loading = false

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
        component.loading = false
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
        // The button should exist.
        const buttonElement = fixture.debugElement.query(By.css('button[label=Checkers]'))
        expect(buttonElement).not.toBeNull()

        // The checker updater shouldn't exist yet.
        let pickerElement = fixture.debugElement.query(By.directive(ConfigCheckerPreferenceUpdaterComponent))
        expect(pickerElement).toBeNull()

        buttonElement.triggerEventHandler('click', new Event('click'))
        fixture.detectChanges()

        // The checker picker should be presented.
        pickerElement = fixture.debugElement.query(By.directive(ConfigCheckerPreferenceUpdaterComponent))
        expect(pickerElement).not.toBeNull()
        const pickerComponent = pickerElement.componentInstance as ConfigCheckerPreferenceUpdaterComponent
        expect(pickerComponent.minimal).toBeTrue()
        expect(pickerComponent.daemonID).not.toBeNull()
    })
})
