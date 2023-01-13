import { ComponentFixture, TestBed, discardPeriodicTasks, fakeAsync, waitForAsync, tick } from '@angular/core/testing'
import { HaStatusComponent } from './ha-status.component'
import { PanelModule } from 'primeng/panel'
import { TooltipModule } from 'primeng/tooltip'
import { MessageModule } from 'primeng/message'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { RouterModule } from '@angular/router'
import { ServicesService, ServicesStatus } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ProgressSpinner, ProgressSpinnerModule } from 'primeng/progressspinner'
import { of, throwError } from 'rxjs'
import { HttpErrorResponse, HttpEvent } from '@angular/common/http'
import { By } from '@angular/platform-browser'

describe('HaStatusComponent', () => {
    let component: HaStatusComponent
    let fixture: ComponentFixture<HaStatusComponent>
    let servicesApi: ServicesService

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [
                PanelModule,
                TooltipModule,
                MessageModule,
                RouterModule,
                HttpClientTestingModule,
                ProgressSpinnerModule,
            ],
            declarations: [HaStatusComponent, LocaltimePipe],
            providers: [ServicesService],
        }).compileComponents()

        servicesApi = TestBed.inject(ServicesService)
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(HaStatusComponent)
        component = fixture.componentInstance
        component.appId = 4
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should present a waiting indicator during loading data', fakeAsync(() => {
        // Mock the API response.
        spyOn(servicesApi, 'getAppServicesStatus').and.returnValue(of({
            items: []
        } as ServicesStatus & HttpEvent<ServicesStatus>))

        // Initially there is no waiting indicator.
        expect(component.loading).toBeFalse()
        let spinner = fixture.debugElement.query(By.directive(ProgressSpinner))
        expect(spinner).toBeNull()

        // Execute ngOnInit hook.
        fixture.detectChanges()
        // Break the interval tasks manually. Otherwise, Jasmine crashes.
        discardPeriodicTasks()

        // Check if the component is in the loading state.
        expect(component.loading).toBeTrue()
        
        // Check if the waiting indicator is presented.
        spinner = fixture.debugElement.query(By.directive(ProgressSpinner))
        expect(spinner).not.toBeNull()
    }))

    it('should present a placeholder when loaded data contain no statuses', fakeAsync(() => {
        // Mock the API response.
        spyOn(servicesApi, 'getAppServicesStatus').and.returnValue(of({
            items: []
        } as ServicesStatus & HttpEvent<ServicesStatus>))

        // Execute ngOnInit hook.
        fixture.detectChanges()
        // Break the interval tasks manually. Otherwise, Jasmine crashes.
        discardPeriodicTasks()

        // Continue the API response processing.
        tick()

        // Check if the data loading is done.
        expect(component.loading).toBeFalse()
        // Render the updated data.
        fixture.detectChanges()
        
        // Check if there is no waiting indicator.
        const spinner = fixture.debugElement.query(By.directive(ProgressSpinner))
        expect(spinner).toBeNull()

        // Check if there is the empty data placeholder.
        expect(fixture.debugElement.nativeElement.textContent).toContain(
            "High Availability is not enabled on this server."
        )
    }))

    it('should present a placeholder on the data loading failure', fakeAsync(() => {
        // Mock the API response.
        spyOn(servicesApi, 'getAppServicesStatus').and.returnValue(
            throwError(new HttpErrorResponse({ status: 500 }))
        )

        // Execute ngOnInit hook.
        fixture.detectChanges()
        // Break the interval tasks manually. Otherwise, Jasmine crashes.
        discardPeriodicTasks()

        // Continue the API response processing.
        tick()

        // Check if the data loading is done.
        expect(component.loading).toBeFalse()
        // Render the updated data.
        fixture.detectChanges()
        
        // Check if there is no waiting indicator.
        const spinner = fixture.debugElement.query(By.directive(ProgressSpinner))
        expect(spinner).toBeNull()

        // Check if there is the empty data placeholder.
        expect(fixture.debugElement.nativeElement.textContent).toContain(
            "High Availability is not enabled on this server."
        )
    }))
})
