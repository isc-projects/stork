import { ComponentFixture, TestBed, fakeAsync, flush, tick } from '@angular/core/testing'
import { FormsModule } from '@angular/forms'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { By } from '@angular/platform-browser'
import { BehaviorSubject, Observable, of, throwError } from 'rxjs'

import { ConfirmationService, Message, MessageService } from 'primeng/api'
import { SelectButtonModule } from 'primeng/selectbutton'
import { TableModule } from 'primeng/table'

import { MachinesPageComponent } from './machines-page.component'
import { AppsVersions, GetMachinesServerToken200Response, Machine, ServicesService, SettingsService } from '../backend'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { DialogModule } from 'primeng/dialog'
import { TabMenuModule } from 'primeng/tabmenu'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { MenuModule } from 'primeng/menu'
import { ProgressBarModule } from 'primeng/progressbar'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { AppDaemonsStatusComponent } from '../app-daemons-status/app-daemons-status.component'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { HttpErrorResponse } from '@angular/common/http'
import anything = jasmine.anything
import { MessagesModule } from 'primeng/messages'
import {
    ActivatedRoute,
    ActivatedRouteSnapshot,
    convertToParamMap,
    NavigationEnd,
    Router,
    RouterModule,
} from '@angular/router'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { Severity, VersionService } from '../version.service'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { MachinesTableComponent } from '../machines-table/machines-table.component'
import { BadgeModule } from 'primeng/badge'
import { PanelModule } from 'primeng/panel'
import { TriStateCheckboxModule } from 'primeng/tristatecheckbox'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { TagModule } from 'primeng/tag'
import createSpyObj = jasmine.createSpyObj
import objectContaining = jasmine.objectContaining

describe('MachinesPageComponent', () => {
    let component: MachinesPageComponent
    let fixture: ComponentFixture<MachinesPageComponent>
    let servicesApi: any
    let msgService: MessageService
    let router: Router
    let route: ActivatedRoute
    let versionServiceStub: Partial<VersionService>
    let routerEventSubject: BehaviorSubject<NavigationEnd>
    let unauthorizedMachinesCountBadge: HTMLElement
    let getSettingsSpy: jasmine.Spy<() => Observable<any>>
    let getMachinesSpy: jasmine.Spy<
        (
            start?: number,
            limit?: number,
            text?: string,
            app?: string,
            authorized?: boolean
        ) => Observable<{ items?: Array<Partial<Machine>>; total?: number }>
    >
    let getMachinesServerTokenSpy: jasmine.Spy<() => Observable<GetMachinesServerToken200Response>>
    let msgSrvAddSpy: jasmine.Spy<(message: Message) => void>

    // prepare responses for api calls
    const getUnauthorizedMachinesResp = {
        items: [
            { hostname: 'aaa', id: 1, address: 'addr1', authorized: false },
            { hostname: 'bbb', id: 2, address: 'addr2', authorized: false },
            { hostname: 'ccc', id: 3, address: 'addr3', authorized: false },
        ],
        total: 3,
    }
    const getAuthorizedMachinesResp = {
        items: [
            { hostname: 'zzz', id: 4, authorized: true },
            { hostname: 'xxx', id: 5, authorized: true },
        ],
        total: 2,
    }
    const getAllMachinesResp = {
        items: [...getUnauthorizedMachinesResp.items, ...getAuthorizedMachinesResp.items],
        total: 5,
    }
    const serverTokenResp = { token: 'ABC' }

    beforeEach(async () => {
        versionServiceStub = {
            sanitizeSemver: () => '1.2.3',
            getCurrentData: () => of({} as AppsVersions),
            getSoftwareVersionFeedback: () => ({ severity: Severity.success, messages: ['test feedback'] }),
        }

        // fake SettingsService
        const registrationEnabled = {
            enableMachineRegistration: true,
        }
        const settingsService = createSpyObj('SettingsService', ['getSettings'])
        getSettingsSpy = settingsService.getSettings.and.returnValue(of(registrationEnabled))

        // fake ServicesService
        servicesApi = createSpyObj('ServicesService', [
            'getMachines',
            'getMachinesServerToken',
            'regenerateMachinesServerToken',
            'getUnauthorizedMachinesCount',
            'updateMachine',
        ])

        getMachinesSpy = servicesApi.getMachines.and.returnValue(of(getAllMachinesResp))
        getMachinesSpy.withArgs(0, 10, null, null, true).and.returnValue(of(getAuthorizedMachinesResp))
        getMachinesSpy.withArgs(0, 10, null, null, false).and.returnValue(of(getUnauthorizedMachinesResp))

        getMachinesServerTokenSpy = servicesApi.getMachinesServerToken.and.returnValue(of(serverTokenResp))
        servicesApi.getUnauthorizedMachinesCount.and.returnValue(of(3))

        await TestBed.configureTestingModule({
            providers: [
                MessageService,
                ConfirmationService,
                { provide: ServicesService, useValue: servicesApi },
                { provide: VersionService, useValue: versionServiceStub },
                { provide: SettingsService, useValue: settingsService },
            ],
            imports: [
                HttpClientTestingModule,
                RouterModule.forRoot([
                    {
                        path: 'machines',
                        pathMatch: 'full',
                        redirectTo: 'machines/all',
                    },
                    {
                        path: 'machines/:id',
                        component: MachinesPageComponent,
                    },
                ]),
                FormsModule,
                SelectButtonModule,
                TableModule,
                DialogModule,
                TabMenuModule,
                MenuModule,
                ProgressBarModule,
                OverlayPanelModule,
                NoopAnimationsModule,
                BreadcrumbModule,
                MessagesModule,
                ConfirmDialogModule,
                BadgeModule,
                PanelModule,
                TriStateCheckboxModule,
                TagModule,
            ],
            declarations: [
                MachinesPageComponent,
                LocaltimePipe,
                PlaceholderPipe,
                BreadcrumbsComponent,
                HelpTipComponent,
                AppDaemonsStatusComponent,
                VersionStatusComponent,
                MachinesTableComponent,
                PluralizePipe,
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(MachinesPageComponent)
        component = fixture.componentInstance
        msgService = fixture.debugElement.injector.get(MessageService)
        fixture.debugElement.injector.get(VersionService)
        route = fixture.debugElement.injector.get(ActivatedRoute)
        route.snapshot = {
            paramMap: convertToParamMap({}),
            queryParamMap: convertToParamMap({}),
        } as ActivatedRouteSnapshot
        router = fixture.debugElement.injector.get(Router)
        routerEventSubject = new BehaviorSubject(new NavigationEnd(1, 'machines', 'machines/all'))
        spyOnProperty(router, 'events').and.returnValue(routerEventSubject)
        msgSrvAddSpy = spyOn(msgService, 'add')

        fixture.detectChanges()
        unauthorizedMachinesCountBadge = fixture.nativeElement.querySelector('div.p-selectbutton span.p-badge')

        // Do not save table state between tests, because that makes tests unstable.
        spyOn(component.machinesTable.table, 'saveState').and.callFake(() => {})

        // Wait until table's data loading is finished.
        await fixture.whenStable()
        fixture.detectChanges()
    })

    /**
     * Triggers the component handler called when the route changes.
     * @param params The parameters to pass to the route.
     * @param queryParams The queryParameters to pass to the route.
     */
    function navigate(
        params: { id?: number | string },
        queryParams?: { authorized?: 'true' | 'false'; text?: string }
    ) {
        route.snapshot = {
            paramMap: convertToParamMap(params),
            queryParamMap: convertToParamMap(queryParams || {}),
        } as ActivatedRouteSnapshot

        const queryParamsList = []
        for (const k in queryParams) {
            if (queryParams[k]) {
                queryParamsList.push(`${encodeURIComponent(k)}=${encodeURIComponent(queryParams[k])}`)
            }
        }

        const eid = routerEventSubject.getValue().id + 1
        routerEventSubject.next(
            new NavigationEnd(
                eid,
                `machines/${params.id}?${queryParamsList.join('&')}`,
                `machines/${params.id}?${queryParamsList.join('&')}`
            )
        )
    }

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should not display agent installation instruction if there is an error in getMachinesServerToken', fakeAsync(() => {
        // dialog should be hidden
        expect(component.displayAgentInstallationInstruction).toBeFalse()

        // prepare error response for call to getMachinesServerToken
        const serverTokenRespErr: any = { statusText: 'some error' }
        getMachinesServerTokenSpy.and.returnValue(throwError(() => serverTokenRespErr))

        const showBtnEl = fixture.debugElement.query(By.css('#show-agent-installation-instruction-button'))
        expect(showBtnEl).toBeTruthy()

        // show instruction but error should appear, so it should be handled
        showBtnEl.nativeElement.click()

        tick() // async message service add()

        // check if it is NOT displayed and server token is still empty
        expect(component.displayAgentInstallationInstruction).toBeFalse()
        expect(getMachinesServerTokenSpy).toHaveBeenCalledTimes(1)
        expect(component.serverToken).toBe('')

        // error message should be issued
        expect(msgSrvAddSpy).toHaveBeenCalledOnceWith(
            objectContaining({ severity: 'error', summary: 'Cannot get server token' })
        )
    }))

    it('should display agent installation instruction if all is ok', async () => {
        // dialog should be hidden
        expect(component.displayAgentInstallationInstruction).toBeFalse()

        const showBtnEl = fixture.debugElement.query(By.css('#show-agent-installation-instruction-button'))
        expect(showBtnEl).toBeTruthy()

        // show instruction
        showBtnEl.triggerEventHandler('click', null)
        await fixture.whenStable()
        fixture.detectChanges()

        // check if it is displayed and server token retrieved
        expect(component.displayAgentInstallationInstruction).toBeTrue()
        expect(getMachinesServerTokenSpy).toHaveBeenCalled()
        expect(component.serverToken).toBe('ABC')

        // regenerate server token
        const regenerateMachinesServerTokenResp: any = { token: 'DEF' }
        servicesApi.regenerateMachinesServerToken.and.returnValue(of(regenerateMachinesServerTokenResp))
        component.regenerateServerToken()
        await fixture.whenStable()
        fixture.detectChanges()

        // check if server token has changed
        expect(component.serverToken).toBe('DEF')

        // close instruction
        const closeBtnEl = fixture.debugElement.query(By.css('#close-agent-installation-instruction-button'))
        expect(closeBtnEl).toBeTruthy()
        closeBtnEl.triggerEventHandler('click', null)

        // now dialog should be hidden
        expect(component.displayAgentInstallationInstruction).toBeFalse()
    })

    it('should error msg if regenerateServerToken fails', async () => {
        // dialog should be hidden
        expect(component.displayAgentInstallationInstruction).toBeFalse()

        const showBtnEl = fixture.debugElement.query(By.css('#show-agent-installation-instruction-button'))
        expect(showBtnEl).toBeTruthy()

        // show instruction but error should appear, so it should be handled
        showBtnEl.nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()

        // check if it is displayed and server token retrieved
        expect(component.displayAgentInstallationInstruction).toBeTrue()
        expect(getMachinesServerTokenSpy).toHaveBeenCalledTimes(1)
        expect(component.serverToken).toBe('ABC')

        // regenerate server token but it returns error, so in UI token should not change
        const regenerateMachinesServerTokenRespErr: any = { statusText: 'some error' }
        servicesApi.regenerateMachinesServerToken.and.returnValue(
            throwError(() => regenerateMachinesServerTokenRespErr)
        )

        const regenerateBtnDe = fixture.debugElement.query(By.css('#regenerate-server-token-button'))
        expect(regenerateBtnDe).toBeTruthy()
        regenerateBtnDe.nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()

        // check if server token has NOT changed
        expect(component.serverToken).toBe('ABC')

        // error message should be issued
        expect(msgSrvAddSpy).toHaveBeenCalledOnceWith(
            objectContaining({ severity: 'error', summary: 'Cannot regenerate server token' })
        )

        // close instruction
        const closeBtnEl = fixture.debugElement.query(By.css('#close-agent-installation-instruction-button'))
        expect(closeBtnEl).toBeTruthy()
        closeBtnEl.nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()

        // now dialog should be hidden
        expect(component.displayAgentInstallationInstruction).toBeFalse()
    })

    it('should list machines', async () => {
        // Data loading should be done by now.
        expect(component.machinesTable.dataLoading).toBeFalse()

        // There is no authorized/unauthorized machines filter applied - all authorized and unauthorized machines are visible.
        expect(component.showAuthorized).toBeNull()
        expect(component.machinesTable.hasPrefilter()).toBeFalse()
        expect(component.machinesTable.hasFilter(component.machinesTable.table)).toBeFalse()
        expect(component.tabs?.[0].routerLink).toBe('/machines/all')
        expect(component.tabs?.[0].queryParams).toBeUndefined()
        expect(component.machinesTable.totalRecords).toBe(5)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(unauthorizedMachinesCountBadge.textContent).toBe('3')

        // Get references to select buttons.
        const selectButtons = fixture.debugElement.queryAll(By.css('#unauthorized-select-button .p-button'))
        expect(selectButtons).toBeTruthy()
        expect(selectButtons.length).toBeGreaterThanOrEqual(2)
        const authSelectBtnEl = selectButtons[0]
        const unauthSelectBtnEl = selectButtons[1]
        expect(authSelectBtnEl).toBeTruthy()
        expect(unauthSelectBtnEl).toBeTruthy()
        expect(authSelectBtnEl.nativeElement.childNodes[0].innerText).toBe('Authorized')
        expect(unauthSelectBtnEl.nativeElement.childNodes[0].innerText).toBe('Unauthorized')
        expect(authSelectBtnEl.nativeElement).withContext('should not be highlighted').not.toHaveClass('p-highlight')
        expect(unauthSelectBtnEl.nativeElement).withContext('should not be highlighted').not.toHaveClass('p-highlight')

        // Navigate to Unauthorized machines only view.
        navigate({ id: 'all' }, { authorized: 'false' })
        await fixture.whenStable()
        fixture.detectChanges()

        // There is unauthorized machines filter applied - only unauthorized machines are visible.
        expect(component.showAuthorized).toBeFalse()
        expect(component.machinesTable.hasPrefilter()).toBeFalse()
        expect(component.machinesTable.hasFilter(component.machinesTable.table)).toBeTrue()
        expect(component.machinesTable.totalRecords).toBe(3)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(unauthorizedMachinesCountBadge.textContent).toBe('3')

        expect(authSelectBtnEl.nativeElement).withContext('should not be highlighted').not.toHaveClass('p-highlight')
        expect(unauthSelectBtnEl.nativeElement).withContext('should be highlighted').toHaveClass('p-highlight')

        // Check if hostnames are displayed.
        const nativeEl = fixture.nativeElement
        expect(nativeEl.textContent).toContain('aaa')
        expect(nativeEl.textContent).toContain('bbb')
        expect(nativeEl.textContent).toContain('ccc')

        // Navigate to Authorized machines only view.
        navigate({ id: 'all' }, { authorized: 'true' })
        await fixture.whenStable()
        fixture.detectChanges()

        // There is authorized machines filter applied - only authorized machines are visible.
        expect(component.showAuthorized).toBeTrue()
        expect(component.machinesTable.hasPrefilter()).toBeFalse()
        expect(component.machinesTable.hasFilter(component.machinesTable.table)).toBeTrue()
        expect(component.machinesTable.totalRecords).toBe(2)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(unauthorizedMachinesCountBadge.textContent).toBe('3')

        expect(authSelectBtnEl.nativeElement).withContext('should be highlighted').toHaveClass('p-highlight')
        expect(unauthSelectBtnEl.nativeElement).withContext('should not be highlighted').not.toHaveClass('p-highlight')

        // Check if hostnames are displayed.
        expect(nativeEl.textContent).toContain('zzz')
        expect(nativeEl.textContent).toContain('xxx')
        expect(nativeEl.textContent).not.toContain('aaa')
    })

    it('should refresh unauthorized machines count', fakeAsync(() => {
        component.machinesTable.unauthorizedMachinesCountChange.emit(4)
        tick()
        fixture.detectChanges()

        expect(component.unauthorizedMachinesCount).toBe(4)
        expect(unauthorizedMachinesCountBadge.textContent).toBe('4')
    }))

    it('should list unauthorized machines requested via URL', fakeAsync(() => {
        // Navigate to Unauthorized machines only view.
        navigate({ id: 'all' }, { authorized: 'false' })
        tick()
        fixture.detectChanges()

        expect(component.showAuthorized).toBeFalse()
    }))

    it('should not list machine as authorized when there was an http status 502 during authorization - bulk authorize - first machine fails', async () => {
        // Navigate to Unauthorized machines only view.
        navigate({ id: 'all' }, { authorized: 'false' })
        await fixture.whenStable()
        fixture.detectChanges()

        // There is unauthorized machines filter applied - only unauthorized machines are visible.
        expect(component.showAuthorized).toBeFalse()
        expect(component.machinesTable.hasPrefilter()).toBeFalse()
        expect(component.machinesTable.hasFilter(component.machinesTable.table)).toBeTrue()

        // check if hostnames are displayed
        const nativeEl = fixture.nativeElement
        expect(nativeEl.textContent).toContain('aaa')
        expect(nativeEl.textContent).toContain('bbb')
        expect(nativeEl.textContent).toContain('ccc')

        // get references to rows' checkboxes
        const checkboxes = fixture.nativeElement.querySelectorAll('table .p-checkbox')
        expect(checkboxes).toBeTruthy()
        expect(checkboxes.length).toBeGreaterThanOrEqual(4)
        // checkboxes[0] is "select all" checkbox, skipped on purpose in this test
        const firstCheckbox = checkboxes[1]
        const secondCheckbox = checkboxes[2]

        // select first two unauthorized machines
        firstCheckbox.dispatchEvent(new Event('click'))
        secondCheckbox.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        // get reference to "Authorize selected" button
        const bulkAuthorizeBtnNodeList = fixture.nativeElement.querySelectorAll('#authorize-selected-button button')
        expect(bulkAuthorizeBtnNodeList).toBeTruthy()
        expect(bulkAuthorizeBtnNodeList.length).toEqual(1)

        const bulkAuthorizeBtn = bulkAuthorizeBtnNodeList[0]
        expect(bulkAuthorizeBtn).toBeTruthy()

        // prepare 502 error response for the first machine of the bulk of machines to be authorized
        const fakeError = new HttpErrorResponse({ status: 502 })
        servicesApi.updateMachine.withArgs(1, anything()).and.returnValue(throwError(() => fakeError))
        servicesApi.updateMachine
            .withArgs(2, anything())
            .and.returnValue(of({ hostname: 'bbb', id: 2, address: 'addr2', authorized: true } as any))

        // click "Authorize selected" button
        bulkAuthorizeBtn.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        // we expect that unauthorized machines list was not changed due to 502 error
        // 'updateMachine' API was called only once for the first machine
        expect(servicesApi.updateMachine).toHaveBeenCalledWith(1, {
            hostname: 'aaa',
            id: 1,
            address: 'addr1',
            authorized: true,
        })
        expect(component.showAuthorized).toBeFalse()
        expect(component.machinesTable.totalRecords).toBe(3)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(unauthorizedMachinesCountBadge.textContent).toBe('3')

        // check if hostnames are displayed
        expect(nativeEl.textContent).toContain('aaa')
        expect(nativeEl.textContent).toContain('bbb')
        expect(nativeEl.textContent).toContain('ccc')

        // Navigate to Authorized machines only view.
        navigate({ id: 'all' }, { authorized: 'true' })
        await fixture.whenStable()
        fixture.detectChanges()

        // There is authorized machines filter applied - only authorized machines are visible.
        expect(component.showAuthorized).toBeTrue()
        expect(component.machinesTable.totalRecords).toBe(2)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(unauthorizedMachinesCountBadge.textContent).toBe('3')

        // Check if hostnames are displayed.
        expect(nativeEl.textContent).toContain('zzz')
        expect(nativeEl.textContent).toContain('xxx')
        expect(nativeEl.textContent).not.toContain('aaa')
        expect(nativeEl.textContent).not.toContain('bbb')
    })

    it('should not list machine as authorized when there was an http status 502 during authorization - bulk authorize - second machine fails', async () => {
        // prepare 502 error response for the second machine of the bulk of machines to be authorized
        // first machine authorization shall succeed, third shall be skipped because it was after the 502 error
        const fakeError = new HttpErrorResponse({ status: 502 })
        servicesApi.updateMachine
            .withArgs(1, anything())
            .and.returnValue(of({ hostname: 'aaa', id: 1, address: 'addr1', authorized: true } as any))
        servicesApi.updateMachine.withArgs(2, anything()).and.returnValue(throwError(() => fakeError))
        servicesApi.updateMachine
            .withArgs(3, anything())
            .and.returnValue(of({ hostname: 'ccc', id: 3, address: 'addr3', authorized: true } as any))

        const getUnauthorizedMachinesRespAfter: any = {
            items: [
                { hostname: 'bbb', id: 2, address: 'addr2' },
                { hostname: 'ccc', id: 3, address: 'addr3' },
            ],
            total: 2,
        }
        const getAuthorizedMachinesRespAfter: any = {
            items: [{ hostname: 'zzz' }, { hostname: 'xxx' }, { hostname: 'aaa', id: 1, address: 'addr1' }],
            total: 3,
        }

        // called one time in loadMachines(event), which lazily loads data for
        // authorized machines table. Text and app filters are undefined.
        getMachinesSpy.withArgs(0, 10, null, null, true).and.returnValue(of(getAuthorizedMachinesRespAfter))

        // Navigate to Unauthorized machines only view.
        navigate({ id: 'all' }, { authorized: 'false' })
        await fixture.whenStable()
        fixture.detectChanges()

        // There is unauthorized machines filter applied - only unauthorized machines are visible.
        expect(component.showAuthorized).toBeFalse()
        expect(component.machinesTable.hasPrefilter()).toBeFalse()
        expect(component.machinesTable.hasFilter(component.machinesTable.table)).toBeTrue()
        expect(component.machinesTable.totalRecords).toBe(3)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(unauthorizedMachinesCountBadge.textContent).toBe('3')

        // check if hostnames are displayed
        const nativeEl = fixture.nativeElement
        expect(nativeEl.textContent).toContain('aaa')
        expect(nativeEl.textContent).toContain('bbb')
        expect(nativeEl.textContent).toContain('ccc')

        // get references to rows' checkboxes
        const checkboxes = fixture.nativeElement.querySelectorAll('table .p-checkbox')
        expect(checkboxes).toBeTruthy()
        expect(checkboxes.length).toBeGreaterThanOrEqual(4)
        const selectAllCheckbox = checkboxes[0]

        // select all unauthorized machines
        selectAllCheckbox.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        // get reference to "Authorize selected" button
        const bulkAuthorizeBtnNodeList = fixture.nativeElement.querySelectorAll('#authorize-selected-button button')
        expect(bulkAuthorizeBtnNodeList).toBeTruthy()
        expect(bulkAuthorizeBtnNodeList.length).toEqual(1)

        const bulkAuthorizeBtn = bulkAuthorizeBtnNodeList[0]
        expect(bulkAuthorizeBtn).toBeTruthy()

        getMachinesSpy.withArgs(0, 10, null, null, false).and.returnValue(of(getUnauthorizedMachinesRespAfter))
        servicesApi.getUnauthorizedMachinesCount.and.returnValue(of(2))

        // click "Authorize selected" button
        bulkAuthorizeBtn.dispatchEvent(new Event('click'))
        await fixture.whenStable()
        fixture.detectChanges()

        // we expect that first machine of the bulk was authorized but second and third were not due to 502 error
        expect(servicesApi.updateMachine).toHaveBeenCalledWith(1, {
            hostname: 'aaa',
            id: 1,
            address: 'addr1',
            authorized: true,
        })
        expect(servicesApi.updateMachine).toHaveBeenCalledWith(2, {
            hostname: 'bbb',
            id: 2,
            address: 'addr2',
            authorized: true,
        })
        // First machine should be authorized and second failed to authorize.
        expect(msgSrvAddSpy).toHaveBeenCalledTimes(2)
        expect(msgSrvAddSpy).toHaveBeenCalledWith(
            objectContaining({ severity: 'success', summary: 'Machine authorized' })
        )
        expect(msgSrvAddSpy).toHaveBeenCalledWith(
            objectContaining({ severity: 'error', summary: 'Machine authorization failed' })
        )
        expect(component.showAuthorized).toBeFalse()
        expect(component.machinesTable.totalRecords).toBe(2)
        expect(component.unauthorizedMachinesCount).toBe(2)
        expect(unauthorizedMachinesCountBadge.textContent).toBe('2')

        // check if hostnames are displayed
        expect(nativeEl.textContent).toContain('bbb')
        expect(nativeEl.textContent).toContain('ccc')
        expect(nativeEl.textContent).not.toContain('aaa')

        // Navigate to Authorized machines only view.
        navigate({ id: 'all' }, { authorized: 'true' })
        await fixture.whenStable()
        fixture.detectChanges()

        expect(servicesApi.getUnauthorizedMachinesCount).toHaveBeenCalled()
        expect(component.showAuthorized).toBeTrue()
        expect(component.machinesTable.totalRecords).toBe(3)
        expect(component.unauthorizedMachinesCount).toBe(2)
        expect(unauthorizedMachinesCountBadge.textContent).toBe('2')

        // check if hostnames are displayed
        expect(nativeEl.textContent).toContain('zzz')
        expect(nativeEl.textContent).toContain('xxx')
        expect(nativeEl.textContent).toContain('aaa')
        expect(nativeEl.textContent).not.toContain('bbb')
        expect(nativeEl.textContent).not.toContain('ccc')
    })

    it('should button menu click trigger the download handler', fakeAsync(() => {
        // Navigate to Authorized machines only view.
        navigate({ id: 'all' }, { authorized: 'true' })
        tick()
        fixture.detectChanges()

        expect(component.showAuthorized).toBeTrue()
        expect(component.machinesTable.totalRecords).toBe(2)

        // Show the menu for the machine with ID=4.
        const menuButton = fixture.debugElement.query(By.css('#show-machines-menu-4'))
        expect(menuButton).not.toBeNull()

        menuButton.nativeElement.click()
        flush()
        fixture.detectChanges()

        // Check the dump button.
        // The menu items don't render the IDs in PrimeNG >= 16.
        const dumpButton = fixture.debugElement.query(By.css('#dump-single-machine a'))
        expect(dumpButton).not.toBeNull()

        const downloadSpy = spyOn(component, 'downloadDump').and.returnValue()

        const dumpButtonElement = dumpButton.nativeElement as HTMLButtonElement
        dumpButtonElement.click()
        flush()
        fixture.detectChanges()

        expect(downloadSpy).toHaveBeenCalledOnceWith(objectContaining({ id: 4 }))
    }))

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('Services')
        expect(breadcrumbsComponent.items[1].label).toEqual('Machines')
    })

    it('should display a warning about disabled registration', fakeAsync(() => {
        // Prepare response for the call to getMachines().
        const getMachinesResp: any = {
            items: [],
            total: 0,
        }
        getMachinesSpy.withArgs(0, 10, null, null, true).and.returnValue(of(getMachinesResp))
        getMachinesSpy.withArgs(0, 10, null, null, false).and.returnValue(of(getMachinesResp))

        // Simulate disabled machine registration.
        const getSettingsResp: any = {
            enableMachineRegistration: false,
        }
        getSettingsSpy.and.returnValue(of(getSettingsResp))

        component.ngOnInit()
        tick()
        fixture.detectChanges()

        // Navigate to Authorized machines only view.
        navigate({ id: 'all' }, { authorized: 'true' })
        tick()
        fixture.detectChanges()

        // Initially, we show authorized machines. In that case we don't show a warning.
        expect(component.showAuthorized).toBeTrue()
        let messages = fixture.debugElement.query(By.css('p-messages'))
        expect(messages).toBeFalsy()

        // Show unauthorized machines.
        navigate({ id: 'all' }, { authorized: 'false' })
        tick()
        fixture.detectChanges()

        expect(component.showAuthorized).toBeFalse()

        // This time we should show the warning that the machines registration is disabled.
        messages = fixture.debugElement.query(By.css('p-messages'))
        expect(messages).toBeTruthy()
        expect(messages.nativeElement.innerText).toContain('Registration of new machines is disabled')
    }))

    it('should not display a warning about disabled registration', fakeAsync(() => {
        // Prepare response for the call to getMachines().
        const getMachinesResp: any = {
            items: [],
            total: 0,
        }
        getMachinesSpy.withArgs(0, 10, null, null, true).and.returnValue(of(getMachinesResp))
        getMachinesSpy.withArgs(0, 10, null, null, false).and.returnValue(of(getMachinesResp))

        // Navigate to Authorized machines only view.
        navigate({ id: 'all' }, { authorized: 'true' })
        tick()
        fixture.detectChanges()

        // Showing authorized machines. The warning is never displayed in such a case.
        expect(component.showAuthorized).toBeTrue()
        let messages = fixture.debugElement.query(By.css('p-messages'))
        expect(messages).toBeFalsy()

        // Show unauthorized machines.
        navigate({ id: 'all' }, { authorized: 'false' })
        tick()
        fixture.detectChanges()

        expect(component.showAuthorized).toBeFalse()

        // The warning should not be displayed because the registration is enabled.
        messages = fixture.debugElement.query(By.css('p-messages'))
        expect(messages).toBeFalsy()
    }))
})
