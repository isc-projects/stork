import { ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing'

import { CommunicationStatusPageComponent } from './communication-status-page.component'
import { RouterTestingModule } from '@angular/router/testing'
import { MessageService } from 'primeng/api'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { TooltipModule } from 'primeng/tooltip'
import { TreeModule } from 'primeng/tree'
import { ButtonModule } from 'primeng/button'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { By } from '@angular/platform-browser'
import { ServicesService } from '../backend'
import { of, throwError } from 'rxjs'
import { CommunicationStatusTreeComponent } from '../communication-status-tree/communication-status-tree.component'

describe('CommunicationStatusPageComponent', () => {
    let component: CommunicationStatusPageComponent
    let fixture: ComponentFixture<CommunicationStatusPageComponent>
    let messageService: MessageService
    let servicesService: ServicesService

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [MessageService],
            imports: [
                BreadcrumbModule,
                ButtonModule,
                HttpClientTestingModule,
                NoopAnimationsModule,
                RouterTestingModule,
                OverlayPanelModule,
                ProgressSpinnerModule,
                TooltipModule,
                TreeModule,
            ],
            declarations: [
                BreadcrumbsComponent,
                CommunicationStatusPageComponent,
                CommunicationStatusTreeComponent,
                HelpTipComponent,
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(CommunicationStatusPageComponent)
        component = fixture.componentInstance
        messageService = fixture.debugElement.injector.get(MessageService)
        servicesService = fixture.debugElement.injector.get(ServicesService)
        fixture.detectChanges()
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
        const helpTip = fixture.debugElement.query(By.css('app-help-tip'))
        expect(helpTip).toBeTruthy()
    })

    it('should reload the tree', fakeAsync(() => {
        let list: any = {
            items: [],
            total: 0,
        }
        spyOn(servicesService, 'getAppsWithCommunicationIssues').and.returnValue(of(list))
        component.onReload()
        tick()
        fixture.detectChanges()
        expect(servicesService.getAppsWithCommunicationIssues).toHaveBeenCalled()

        const issuesTree = fixture.debugElement.query(By.css('app-communication-status-tree'))
        expect(issuesTree).toBeTruthy()
        expect(issuesTree.nativeElement.innerText).toContain('No communication issues found.')
    }))

    it('should show an error when getting communication issues fails', fakeAsync(() => {
        spyOn(messageService, 'add')
        spyOn(servicesService, 'getAppsWithCommunicationIssues').and.returnValue(throwError({ status: 404 }))
        component.ngOnInit()
        tick()
        expect(messageService.add).toHaveBeenCalled()
    }))

    it('should show the progress spinner during loading', () => {
        component.loading = true
        const spinner = fixture.debugElement.query(By.css('p-progressSpinner'))
        expect(spinner).toBeTruthy()
    })
})
