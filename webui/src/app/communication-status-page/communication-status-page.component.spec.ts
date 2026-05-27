import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing'

import { CommunicationStatusPageComponent } from './communication-status-page.component'
import { MessageService } from 'primeng/api'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { By } from '@angular/platform-browser'
import { ServicesService } from '../backend'
import { of, throwError } from 'rxjs'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { provideRouter } from '@angular/router'

describe('CommunicationStatusPageComponent', () => {
    let component: CommunicationStatusPageComponent
    let fixture: ComponentFixture<CommunicationStatusPageComponent>
    let messageService: MessageService
    let servicesService: ServicesService

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [
                MessageService,
                provideNoopAnimations(),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideRouter([]),
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(CommunicationStatusPageComponent)
        component = fixture.componentInstance
        messageService = fixture.debugElement.injector.get(MessageService)
        servicesService = fixture.debugElement.injector.get(ServicesService)
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should contain breadcrumbs', () => {
        expect(component.breadcrumbs.length).toBe(2)
        expect(component.breadcrumbs[0]?.label).toBe('Monitoring')
        expect(component.breadcrumbs[1]?.label).toBe('Communication')
    })

    it('should contain page help tip', () => {
        fixture.detectChanges()
        const helpTip = fixture.debugElement.query(By.css('app-help-tip'))
        expect(helpTip).toBeTruthy()
    })

    it('should reload the tree', fakeAsync(() => {
        let list: any = {
            items: [],
            total: 0,
        }
        spyOn(servicesService, 'getDaemonsWithCommunicationIssues').and.returnValue(of(list))
        fixture.detectChanges()
        tick()

        component.onReload()
        tick()
        expect(servicesService.getDaemonsWithCommunicationIssues).toHaveBeenCalledTimes(2)
        expect(component.loading).toBeFalse()
        expect(component.daemons).toEqual([])
    }))

    it('should show an error when getting communication issues fails', fakeAsync(() => {
        spyOn(messageService, 'add')
        spyOn(servicesService, 'getDaemonsWithCommunicationIssues').and.returnValue(throwError({ status: 404 }))
        component.ngOnInit()
        tick()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should show the progress spinner during loading', () => {
        component.loading = true
        fixture.detectChanges()
        const spinner = fixture.debugElement.query(By.css('p-progressSpinner'))
        expect(spinner).toBeTruthy()
    })
})
