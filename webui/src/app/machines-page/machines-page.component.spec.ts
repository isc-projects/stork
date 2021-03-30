import { async, ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing'
import { FormsModule } from '@angular/forms'
import { ActivatedRoute, Router, convertToParamMap } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { By } from '@angular/platform-browser'
import { of, throwError } from 'rxjs'

import { MessageService } from 'primeng/api'
import { SelectButtonModule } from 'primeng/selectbutton'
import { TableModule } from 'primeng/table'

import { MachinesPageComponent } from './machines-page.component'
import { ServicesService, UsersService } from '../backend'
import { LocaltimePipe } from '../localtime.pipe'

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
            ],
            declarations: [MachinesPageComponent, LocaltimePipe],
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

    it('should display agent installation instruction if all is ok', () => {
        // dialog should be hidden
        expect(component.displayAgentInstallationInstruction).toBeFalse()

        // prepare response for call to getMachinesServerToken
        const serverTokenResp: any = { token: 'ABC' }
        spyOn(servicesApi, 'getMachinesServerToken').and.returnValues(of(serverTokenResp))

        const showBtnEl = fixture.debugElement.query(By.css('#show-agent-installation-instruction-button'))

        // show instruction
        showBtnEl.triggerEventHandler('click', null)

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
        closeBtnEl.triggerEventHandler('click', null)

        // now dialog should be hidden
        expect(component.displayAgentInstallationInstruction).toBeFalse()
    })

    it('should error msg if regenerateServerToken fails', () => {
        // dialog should be hidden
        expect(component.displayAgentInstallationInstruction).toBeFalse()

        // prepare response for call to getMachinesServerToken
        const serverTokenResp: any = { token: 'ABC' }
        spyOn(servicesApi, 'getMachinesServerToken').and.returnValue(of(serverTokenResp))

        const showBtnEl = fixture.debugElement.query(By.css('#show-agent-installation-instruction-button'))

        // show instruction but error should appear, so it should be handled
        showBtnEl.triggerEventHandler('click', null)

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
        closeBtnEl.triggerEventHandler('click', null)

        // now dialog should be hidden
        expect(component.displayAgentInstallationInstruction).toBeFalse()
    })

    it('should list machines', fakeAsync(() => {
        expect(component.showUnauthorized).toBeFalse()

        // get references to select buttons
        const selectBtns = fixture.nativeElement.querySelectorAll('#unauthorized-select-button .ui-button')
        const authSelectBtnEl = selectBtns[0]
        const unauthSelectBtnEl = selectBtns[1]

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

        // check if hostnames are displayed
        expect(nativeEl.textContent).toContain('zzz')
        expect(nativeEl.textContent).toContain('xxx')
        expect(nativeEl.textContent).not.toContain('aaa')
    }))
})
