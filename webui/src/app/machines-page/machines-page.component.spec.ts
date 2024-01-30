import { ComponentFixture, TestBed, fakeAsync, flush, tick } from '@angular/core/testing'
import { FormsModule } from '@angular/forms'
import { RouterTestingModule } from '@angular/router/testing'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { By } from '@angular/platform-browser'
import { of, throwError } from 'rxjs'

import { MessageService } from 'primeng/api'
import { SelectButtonModule } from 'primeng/selectbutton'
import { TableModule } from 'primeng/table'

import { MachinesPageComponent } from './machines-page.component'
import { ServicesService, UsersService } from '../backend'
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

describe('MachinesPageComponent', () => {
    let component: MachinesPageComponent
    let fixture: ComponentFixture<MachinesPageComponent>
    let servicesApi: ServicesService
    let msgService: MessageService

    beforeEach(fakeAsync(() => {
        TestBed.configureTestingModule({
            providers: [MessageService, ServicesService, UsersService],
            imports: [
                HttpClientTestingModule,
                RouterTestingModule.withRoutes([{ path: 'machines/all', component: MachinesPageComponent }]),
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

        // check if hostnames are displayed
        expect(nativeEl.textContent).toContain('zzz')
        expect(nativeEl.textContent).toContain('xxx')
        expect(nativeEl.textContent).not.toContain('aaa')
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
})
