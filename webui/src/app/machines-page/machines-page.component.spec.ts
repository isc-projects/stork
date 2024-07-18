import { ComponentFixture, TestBed, fakeAsync, flush, tick } from '@angular/core/testing'
import { FormsModule } from '@angular/forms'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { By } from '@angular/platform-browser'
import { of, throwError } from 'rxjs'

import { MessageService } from 'primeng/api'
import { SelectButtonModule } from 'primeng/selectbutton'
import { TableModule } from 'primeng/table'

import { MachinesPageComponent } from './machines-page.component'
import { ServicesService, SettingsService, UsersService } from '../backend'
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
import { RouterModule } from '@angular/router'
import { HttpErrorResponse } from '@angular/common/http'
import anything = jasmine.anything
import { MessagesModule } from 'primeng/messages'
import { ActivatedRoute, convertToParamMap, Router, RouterModule } from '@angular/router'

describe('MachinesPageComponent', () => {
    let component: MachinesPageComponent
    let fixture: ComponentFixture<MachinesPageComponent>
    let servicesApi: ServicesService
    let msgService: MessageService
    let settingsService: SettingsService
    let router: Router
    let route: ActivatedRoute

    beforeEach(fakeAsync(() => {
        TestBed.configureTestingModule({
            providers: [MessageService, ServicesService, UsersService],
            imports: [
                HttpClientTestingModule,
                RouterModule.forRoot([
                    {
                        path: 'machines/unauthorized',
                        component: MachinesPageComponent,
                    },
                    {
                        path: 'machines/authorized',
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
            ],
            declarations: [
                MachinesPageComponent,
                LocaltimePipe,
                PlaceholderPipe,
                BreadcrumbsComponent,
                HelpTipComponent,
                AppDaemonsStatusComponent,
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(MachinesPageComponent)
        component = fixture.componentInstance
        servicesApi = fixture.debugElement.injector.get(ServicesService)
        msgService = fixture.debugElement.injector.get(MessageService)
        settingsService = fixture.debugElement.injector.get(SettingsService)
        router = fixture.debugElement.injector.get(Router)
        route = fixture.debugElement.injector.get(ActivatedRoute)

        fixture.detectChanges()
        tick()
    }))

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should not display agent installation instruction if there is an error in getMachinesServerToken', () => {
        const msgSrvAddSpy = spyOn(msgService, 'add')

        // dialog should be hidden
        expect(component.displayAgentInstallationInstruction).toBeFalse()

        // prepare error response for call to getMachinesServerToken
        const serverTokenRespErr: any = { statusText: 'some error' }
        spyOn(servicesApi, 'getMachinesServerToken').and.returnValue(throwError(serverTokenRespErr))

        const showBtnEl = fixture.debugElement.query(By.css('#show-agent-installation-instruction-button'))
        expect(showBtnEl).toBeDefined()

        // show instruction but error should appear, so it should be handled
        showBtnEl.triggerEventHandler('click', null)

        // check if it is NOT displayed and server token is still empty
        expect(component.displayAgentInstallationInstruction).toBeFalse()
        expect(servicesApi.getMachinesServerToken).toHaveBeenCalled()
        expect(component.serverToken).toBe('')

        // error message should be issued
        expect(msgSrvAddSpy.calls.count()).toBe(1)
        expect(msgSrvAddSpy.calls.argsFor(0)[0]['severity']).toBe('error')
    })

    it('should display agent installation instruction if all is ok', async () => {
        // dialog should be hidden
        expect(component.displayAgentInstallationInstruction).toBeFalse()

        // prepare response for call to getMachinesServerToken
        const serverTokenResp: any = { token: 'ABC' }
        spyOn(servicesApi, 'getMachinesServerToken').and.returnValues(of(serverTokenResp))

        const showBtnEl = fixture.debugElement.query(By.css('#show-agent-installation-instruction-button'))

        // show instruction
        showBtnEl.triggerEventHandler('click', null)
        await fixture.whenStable()
        fixture.detectChanges()

        // check if it is displayed and server token retrieved
        expect(component.displayAgentInstallationInstruction).toBeTrue()
        expect(servicesApi.getMachinesServerToken).toHaveBeenCalled()
        expect(component.serverToken).toBe('ABC')

        // regenerate server token
        const regenerateMachinesServerTokenResp: any = { token: 'DEF' }
        const regenSpy = spyOn(servicesApi, 'regenerateMachinesServerToken')
        regenSpy.and.returnValue(of(regenerateMachinesServerTokenResp))
        component.regenerateServerToken()

        // check if server token has changed
        expect(component.serverToken).toBe('DEF')

        // close instruction
        const closeBtnEl = fixture.debugElement.query(By.css('#close-agent-installation-instruction-button'))
        expect(closeBtnEl).toBeDefined()
        closeBtnEl.triggerEventHandler('click', null)

        // now dialog should be hidden
        expect(component.displayAgentInstallationInstruction).toBeFalse()
    })

    it('should error msg if regenerateServerToken fails', async () => {
        // dialog should be hidden
        expect(component.displayAgentInstallationInstruction).toBeFalse()

        // prepare response for call to getMachinesServerToken
        const serverTokenResp: any = { token: 'ABC' }
        spyOn(servicesApi, 'getMachinesServerToken').and.returnValue(of(serverTokenResp))

        const showBtnEl = fixture.debugElement.query(By.css('#show-agent-installation-instruction-button'))

        // show instruction but error should appear, so it should be handled
        showBtnEl.triggerEventHandler('click', null)
        await fixture.whenStable()
        fixture.detectChanges()

        // check if it is displayed and server token retrieved
        expect(component.displayAgentInstallationInstruction).toBeTrue()
        expect(servicesApi.getMachinesServerToken).toHaveBeenCalled()
        expect(component.serverToken).toBe('ABC')

        const msgSrvAddSpy = spyOn(msgService, 'add')

        // regenerate server token but it returns error, so in UI token should not change
        const regenerateMachinesServerTokenRespErr: any = { statusText: 'some error' }
        const regenSpy = spyOn(servicesApi, 'regenerateMachinesServerToken')
        regenSpy.and.returnValue(throwError(regenerateMachinesServerTokenRespErr))
        component.regenerateServerToken()

        // check if server token has NOT changed
        expect(component.serverToken).toBe('ABC')

        // error message should be issued
        expect(msgSrvAddSpy.calls.count()).toBe(1)
        expect(msgSrvAddSpy.calls.argsFor(0)[0]['severity']).toBe('error')

        // close instruction
        const closeBtnEl = fixture.debugElement.query(By.css('#close-agent-installation-instruction-button'))
        expect(closeBtnEl).toBeDefined()
        closeBtnEl.triggerEventHandler('click', null)

        // now dialog should be hidden
        expect(component.displayAgentInstallationInstruction).toBeFalse()
    })

    it('should list machines', fakeAsync(() => {
        expect(component.showUnauthorized).toBeFalse()
        expect(component.tabs?.[0].routerLink).toBe('/machines/authorized')

        // get references to select buttons
        const selectButtons = fixture.nativeElement.querySelectorAll('#unauthorized-select-button .p-button')
        const authSelectBtnEl = selectButtons[0]
        const unauthSelectBtnEl = selectButtons[1]

        // prepare response for call to getMachines for (un)authorized machines
        const getUnauthorizedMachinesResp: any = {
            items: [{ hostname: 'aaa' }, { hostname: 'bbb' }, { hostname: 'ccc' }],
            total: 3,
        }
        const getAuthorizedMachinesResp: any = { items: [{ hostname: 'zzz' }, { hostname: 'xxx' }], total: 2 }
        const gmSpy = spyOn(servicesApi, 'getMachines')
        gmSpy.withArgs(0, 1, null, null, false).and.returnValue(of(getUnauthorizedMachinesResp))
        gmSpy.withArgs(0, 10, undefined, undefined, false).and.returnValue(of(getUnauthorizedMachinesResp))
        gmSpy.withArgs(0, 10, undefined, undefined, true).and.returnValue(of(getAuthorizedMachinesResp))

        // show unauthorized machines
        unauthSelectBtnEl.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        expect(component.showUnauthorized).toBeTrue()
        expect(component.totalMachines).toBe(3)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(component.viewSelectionOptions[1].label).toBe('Unauthorized (3)')
        expect(component.tabs?.[0].routerLink).toBe('/machines/unauthorized')

        // check if hostnames are displayed
        const nativeEl = fixture.nativeElement
        expect(nativeEl.textContent).toContain('aaa')
        expect(nativeEl.textContent).toContain('bbb')
        expect(nativeEl.textContent).toContain('ccc')

        // show authorized machines
        authSelectBtnEl.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        expect(component.showUnauthorized).toBeFalse()
        expect(component.totalMachines).toBe(2)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(component.viewSelectionOptions[1].label).toBe('Unauthorized (3)')
        expect(component.tabs?.[0].routerLink).toBe('/machines/authorized')

        // check if hostnames are displayed
        expect(nativeEl.textContent).toContain('zzz')
        expect(nativeEl.textContent).toContain('xxx')
        expect(nativeEl.textContent).not.toContain('aaa')
    }))

    it('should refresh unauthorized machines count', fakeAsync(() => {
        spyOn(servicesApi, 'getUnauthorizedMachinesCount').and.returnValue(of(4 as any))
        tick()
        fixture.detectChanges()
        component.refreshUnauthorizedMachinesCount()

        expect(component.unauthorizedMachinesCount).toBe(4)
        expect(component.viewSelectionOptions[1].label).toBe('Unauthorized (4)')
    }))

    it('should list unauthorized machines requested via URL', fakeAsync(() => {
        router.navigate(['/machines/unauthorized'])
        const paramMap = convertToParamMap({
            id: 'unauthorized',
        })
        spyOnProperty(route, 'paramMap').and.returnValue(of(paramMap))

        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(component.showUnauthorized).toBeTrue()
    }))

    it('should not list machine as authorized when there was an http status 502 during authorization - bulk authorize - first machine fails', fakeAsync(() => {
        expect(component.showUnauthorized).toBeFalse()

        // get references to select buttons
        const selectButtons = fixture.nativeElement.querySelectorAll('#unauthorized-select-button .p-button')
        const authSelectBtnEl = selectButtons[0]
        const unauthSelectBtnEl = selectButtons[1]

        // prepare response for call to getMachines for (un)authorized machines
        const getUnauthorizedMachinesResp: any = {
            items: [
                { hostname: 'aaa', id: 1, address: 'addr1' },
                { hostname: 'bbb', id: 2, address: 'addr2' },
                { hostname: 'ccc', id: 3, address: 'addr3' },
            ],
            total: 3,
        }
        const getAuthorizedMachinesResp: any = { items: [{ hostname: 'zzz' }, { hostname: 'xxx' }], total: 2 }
        const gmSpy = spyOn(servicesApi, 'getMachines')

        // Prepare response for getMachines API being called by refreshUnauthorizedMachinesCount().
        // refreshUnauthorizedMachinesCount() asks for details of only one unauthorized machine.
        // The total unauthorized machines count is essential information here, not the detailed items themselves.
        gmSpy.withArgs(0, 1, null, null, false).and.returnValue(
            of({
                items: [{ hostname: 'aaa', id: 1, address: 'addr1' }],
                total: 3,
            } as any)
        )
        // Prepare response for getMachines API being called by loadMachines(), which lazily loads data for
        // unauthorized machines table. Text and app filters are undefined.
        gmSpy.withArgs(0, 10, undefined, undefined, false).and.returnValue(of(getUnauthorizedMachinesResp))
        // Prepare response for getMachines API being called by loadMachines(), which lazily loads data for
        // authorized machines table. Text and app filters are undefined.
        gmSpy.withArgs(0, 10, undefined, undefined, true).and.returnValue(of(getAuthorizedMachinesResp))

        // show unauthorized machines
        unauthSelectBtnEl.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        expect(component.showUnauthorized).toBeTrue()
        expect(component.totalMachines).toBe(3)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(component.viewSelectionOptions[1].label).toBe('Unauthorized (3)')

        // check if hostnames are displayed
        const nativeEl = fixture.nativeElement
        expect(nativeEl.textContent).toContain('aaa')
        expect(nativeEl.textContent).toContain('bbb')
        expect(nativeEl.textContent).toContain('ccc')

        // get references to rows' checkboxes
        const checkboxes = fixture.nativeElement.querySelectorAll('.p-checkbox')
        expect(checkboxes).toBeTruthy()
        expect(checkboxes.length).toBeGreaterThanOrEqual(3)
        // checkboxes[0] is "select all" checkbox, skipped on purpose in this test
        const firstCheckbox = checkboxes[1]
        const secondCheckbox = checkboxes[2]

        // select first two unauthorized machines
        firstCheckbox.dispatchEvent(new Event('click'))
        secondCheckbox.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        // get reference to "Authorize selected" button
        const bulkAuthorizeBtnNodeList = fixture.nativeElement.querySelectorAll('#authorize-selected-button')
        expect(bulkAuthorizeBtnNodeList).toBeTruthy()
        expect(bulkAuthorizeBtnNodeList.length).toEqual(1)

        const bulkAuthorizeBtn = bulkAuthorizeBtnNodeList[0]
        expect(bulkAuthorizeBtn).toBeTruthy()

        // prepare 502 error response for the first machine of the bulk of machines to be authorized
        const umSpy = spyOn(servicesApi, 'updateMachine')
        const fakeError = new HttpErrorResponse({ status: 502 })
        umSpy.withArgs(1, anything()).and.returnValue(throwError(() => fakeError))
        umSpy
            .withArgs(2, anything())
            .and.returnValue(of({ hostname: 'bbb', id: 2, address: 'addr2', authorized: true } as any))

        // click "Authorize selected" button
        bulkAuthorizeBtn.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        // we expect that unauthorized machines list was not changed due to 502 error
        // 'updateMachine' API was called only once for the first machine
        expect(umSpy).toHaveBeenCalledWith(1, { hostname: 'aaa', id: 1, address: 'addr1', authorized: true })
        expect(component.showUnauthorized).toBeTrue()
        expect(component.totalMachines).toBe(3)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(component.viewSelectionOptions[1].label).toBe('Unauthorized (3)')

        // check if hostnames are displayed
        expect(nativeEl.textContent).toContain('aaa')
        expect(nativeEl.textContent).toContain('bbb')
        expect(nativeEl.textContent).toContain('ccc')

        // show authorized machines
        authSelectBtnEl.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        expect(component.showUnauthorized).toBeFalse()
        expect(component.totalMachines).toBe(2)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(component.viewSelectionOptions[1].label).toBe('Unauthorized (3)')

        // check if hostnames are displayed
        expect(nativeEl.textContent).toContain('zzz')
        expect(nativeEl.textContent).toContain('xxx')
        expect(nativeEl.textContent).not.toContain('aaa')
        expect(nativeEl.textContent).not.toContain('bbb')
    }))

    it('should not list machine as authorized when there was an http status 502 during authorization - bulk authorize - second machine fails', fakeAsync(() => {
        expect(component.showUnauthorized).toBeFalse()

        // get references to select buttons
        const selectButtons = fixture.nativeElement.querySelectorAll('#unauthorized-select-button .p-button')
        const authSelectBtnEl = selectButtons[0]
        const unauthSelectBtnEl = selectButtons[1]

        // prepare response for call to getMachines for (un)authorized machines
        const getUnauthorizedMachinesRespBefore: any = {
            items: [
                { hostname: 'aaa', id: 1, address: 'addr1' },
                { hostname: 'bbb', id: 2, address: 'addr2' },
                { hostname: 'ccc', id: 3, address: 'addr3' },
            ],
            total: 3,
        }
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

        const gmSpy = spyOn(servicesApi, 'getMachines')

        // this is only called once after authorizing selected machines
        // and after switching back to authorized machines view; in refreshUnauthorizedMachinesCount().
        // refreshUnauthorizedMachinesCount() asks for details of only one unauthorized machine.
        // The total unauthorized machines count is essential information here, not the detailed items themselves.
        gmSpy.withArgs(0, 1, null, null, false).and.returnValue(
            of({
                items: [{ hostname: 'bbb', id: 2, address: 'addr2' }],
                total: 2,
            } as any)
        )

        // called two times in loadMachines(event), which lazily loads data for
        // unauthorized machines table. Text and app filters are undefined.
        gmSpy
            .withArgs(0, 10, undefined, undefined, false)
            .and.returnValues(of(getUnauthorizedMachinesRespBefore), of(getUnauthorizedMachinesRespAfter))

        // called one time in loadMachines(event), which lazily loads data for
        // authorized machines table. Text and app filters are undefined.
        gmSpy.withArgs(0, 10, undefined, undefined, true).and.returnValue(of(getAuthorizedMachinesRespAfter))

        // show unauthorized machines
        unauthSelectBtnEl.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        expect(component.showUnauthorized).toBeTrue()
        expect(component.totalMachines).toBe(3)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(component.viewSelectionOptions[1].label).toBe('Unauthorized (3)')

        // check if hostnames are displayed
        const nativeEl = fixture.nativeElement
        expect(nativeEl.textContent).toContain('aaa')
        expect(nativeEl.textContent).toContain('bbb')
        expect(nativeEl.textContent).toContain('ccc')

        // get references to rows' checkboxes
        const checkboxes = fixture.nativeElement.querySelectorAll('.p-checkbox')
        expect(checkboxes).toBeTruthy()
        expect(checkboxes.length).toBeGreaterThanOrEqual(1)
        const selectAllCheckbox = checkboxes[0]

        // select all unauthorized machines
        selectAllCheckbox.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        // get reference to "Authorize selected" button
        const bulkAuthorizeBtnNodeList = fixture.nativeElement.querySelectorAll('#authorize-selected-button')
        expect(bulkAuthorizeBtnNodeList).toBeTruthy()
        expect(bulkAuthorizeBtnNodeList.length).toEqual(1)

        const bulkAuthorizeBtn = bulkAuthorizeBtnNodeList[0]
        expect(bulkAuthorizeBtn).toBeTruthy()

        // prepare 502 error response for the second machine of the bulk of machines to be authorized
        // first machine authorization shall succeed, third shall be skipped because it was after the 502 error
        const umSpy = spyOn(servicesApi, 'updateMachine')
        const fakeError = new HttpErrorResponse({ status: 502 })
        umSpy
            .withArgs(1, anything())
            .and.returnValue(of({ hostname: 'aaa', id: 1, address: 'addr1', authorized: true } as any))
        umSpy.withArgs(2, anything()).and.returnValue(throwError(() => fakeError))
        umSpy
            .withArgs(3, anything())
            .and.returnValue(of({ hostname: 'ccc', id: 3, address: 'addr3', authorized: true } as any))

        // click "Authorize selected" button
        bulkAuthorizeBtn.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        // we expect that first machine of the bulk was authorized but second and third were not due to 502 error
        expect(umSpy).toHaveBeenCalledWith(1, { hostname: 'aaa', id: 1, address: 'addr1', authorized: true })
        expect(umSpy).toHaveBeenCalledWith(2, { hostname: 'bbb', id: 2, address: 'addr2', authorized: true })

        expect(component.showUnauthorized).toBeTrue()
        expect(component.totalMachines).toBe(2)
        expect(component.unauthorizedMachinesCount).toBe(2)
        expect(component.viewSelectionOptions[1].label).toBe('Unauthorized (2)')

        // check if hostnames are displayed
        expect(nativeEl.textContent).toContain('bbb')
        expect(nativeEl.textContent).toContain('ccc')

        // show authorized machines
        authSelectBtnEl.dispatchEvent(new Event('click'))
        fixture.detectChanges()

        expect(gmSpy).toHaveBeenCalledTimes(4)

        expect(component.showUnauthorized).toBeFalse()
        expect(component.totalMachines).toBe(3)
        expect(component.unauthorizedMachinesCount).toBe(2)
        expect(component.viewSelectionOptions[1].label).toBe('Unauthorized (2)')

        // check if hostnames are displayed
        expect(nativeEl.textContent).toContain('zzz')
        expect(nativeEl.textContent).toContain('xxx')
        expect(nativeEl.textContent).toContain('aaa')
        expect(nativeEl.textContent).not.toContain('bbb')
        expect(nativeEl.textContent).not.toContain('ccc')
    }))

    it('should button menu click trigger the download handler', fakeAsync(() => {
        // Prepare the data
        const getAuthorizedMachinesResp: any = {
            items: [
                { id: 1, hostname: 'zzz' },
                { id: 2, hostname: 'xxx' },
            ],
            total: 2,
        }
        spyOn(servicesApi, 'getMachines').and.returnValue(of(getAuthorizedMachinesResp))
        // ngOnInit was already called before we prepared the static response.
        // We have to reload the machines list manually.
        component.loadMachines({ first: 0, rows: 0, filters: {} })
        flush()
        fixture.detectChanges()

        // Show the menu.
        const menuButton = fixture.debugElement.query(By.css('#show-machines-menu'))
        expect(menuButton).not.toBeNull()

        menuButton.triggerEventHandler('click', { currentTarget: menuButton.nativeElement })
        flush()
        fixture.detectChanges()

        // Check the dump button.
        // The menu items don't render the IDs in PrimeNG >= 16.
        const dumpButton = fixture.debugElement.query(
            By.css('[title="Download data archive for troubleshooting purposes"]')
        )
        expect(dumpButton).not.toBeNull()

        const downloadSpy = spyOn(component, 'downloadDump').and.returnValue()

        const dumpButtonElement = dumpButton.nativeElement as HTMLButtonElement
        dumpButtonElement.click()
        flush()
        fixture.detectChanges()

        expect(downloadSpy).toHaveBeenCalledTimes(1)
        expect(downloadSpy.calls.first().args[0].id).toBe(1)
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

    it('should display status of all daemons from all applications', async () => {
        // Prepare the data
        const getAuthorizedMachinesResp: any = {
            items: [
                {
                    id: 1,
                    hostname: 'zzz',
                    apps: [
                        {
                            id: 1,
                            name: 'kea@localhost',
                            type: 'kea',
                            details: {
                                daemons: [
                                    {
                                        active: true,
                                        extendedVersion: '2.2.0',
                                        id: 1,
                                        name: 'dhcp4',
                                    },
                                    {
                                        active: false,
                                        extendedVersion: '2.3.0',
                                        id: 2,
                                        name: 'ca',
                                    },
                                ],
                            },
                        },
                        {
                            id: 2,
                            name: 'bind9@localhost',
                            type: 'bind9',
                            details: {
                                daemon: {
                                    active: true,
                                    id: 3,
                                    name: 'named',
                                },
                            },
                        },
                    ],
                },
            ],
            total: 1,
        }
        spyOn(servicesApi, 'getMachines').and.returnValue(of(getAuthorizedMachinesResp))
        // ngOnInit was already called before we prepared the static response.
        // We have to reload the machines list manually.
        component.loadMachines({ first: 0, rows: 0, filters: {} })

        await fixture.whenStable()
        fixture.detectChanges()

        const textContent = (fixture.debugElement.nativeElement as HTMLElement).textContent

        expect(textContent).toContain('DHCPv4')
        expect(textContent).toContain('CA')
        expect(textContent).toContain('named')
    })

    it('should display a warning about disabled registration', fakeAsync(() => {
        // Get references to select buttons
        const selectButtons = fixture.nativeElement.querySelectorAll('#unauthorized-select-button .p-button')
        const unauthSelectBtnEl = selectButtons[1]

        // Prepare response for the call to getMachines().
        const getMachinesResp: any = {
            items: [],
            total: 0,
        }
        spyOn(servicesApi, 'getMachines').and.returnValue(of(getMachinesResp))

        // Simulate disabled machine registration.
        const getSettingsResp: any = {
            enableMachineRegistration: false,
        }
        spyOn(settingsService, 'getSettings').and.returnValue(of(getSettingsResp))

        component.ngOnInit()
        tick()

        // Initially, we show authorized machines. In that case we don't show a warning.
        expect(component.showUnauthorized).toBeFalse()
        let messages = fixture.debugElement.query(By.css('p-messages'))
        expect(messages).toBeFalsy()

        // Show unauthorized machines.
        unauthSelectBtnEl.dispatchEvent(new Event('click'))
        fixture.detectChanges()
        expect(component.showUnauthorized).toBeTrue()

        // This time we should show the warning that the machines registration is disabled.
        messages = fixture.debugElement.query(By.css('p-messages'))
        expect(messages).toBeTruthy()
        expect(messages.nativeElement.innerText).toContain('Registration of the new machines is disabled')
    }))

    it('should not display a warning about disabled registration', fakeAsync(() => {
        // Get references to select buttons
        const selectButtons = fixture.nativeElement.querySelectorAll('#unauthorized-select-button .p-button')
        const unauthSelectBtnEl = selectButtons[1]

        // Prepare response for the call to getMachines().
        const getMachinesResp: any = {
            items: [],
            total: 0,
        }
        spyOn(servicesApi, 'getMachines').and.returnValue(of(getMachinesResp))

        const getSettingsResp: any = {
            enableMachineRegistration: true,
        }
        spyOn(settingsService, 'getSettings').and.returnValue(of(getSettingsResp))

        component.ngOnInit()
        tick()

        // Showing authorized machines. The warning is never displayed in such a case.
        expect(component.showUnauthorized).toBeFalse()
        let messages = fixture.debugElement.query(By.css('p-messages'))
        expect(messages).toBeFalsy()

        // Show unauthorized machines.
        unauthSelectBtnEl.dispatchEvent(new Event('click'))
        fixture.detectChanges()
        expect(component.showUnauthorized).toBeTrue()

        // The warning should not be displayed because the registration is enabled.
        messages = fixture.debugElement.query(By.css('p-messages'))
        expect(messages).toBeFalsy()
    }))
})
