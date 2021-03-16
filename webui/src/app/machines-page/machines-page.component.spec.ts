import { async, ComponentFixture, TestBed, fakeAsync, tick } from '@angular/core/testing'

import { MachinesPageComponent } from './machines-page.component'
import { ActivatedRoute, Router, convertToParamMap } from '@angular/router'
import { RouterTestingModule } from '@angular/router/testing'
import { ServicesService, UsersService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { SelectButtonModule } from 'primeng/selectbutton'
import { TableModule } from 'primeng/table'
import { By } from '@angular/platform-browser'
import { of } from 'rxjs'

describe('MachinesPageComponent', () => {
    let component: MachinesPageComponent
    let fixture: ComponentFixture<MachinesPageComponent>
    let servicesApi: ServicesService

    beforeEach(fakeAsync(() => {
        TestBed.configureTestingModule({
            providers: [MessageService, ServicesService, UsersService],
            imports: [
                HttpClientTestingModule,
                RouterTestingModule.withRoutes([{ path: 'machines/all', component: MachinesPageComponent }]),
                SelectButtonModule,
                TableModule,
            ],
            declarations: [MachinesPageComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(MachinesPageComponent)
        component = fixture.componentInstance
        servicesApi = fixture.debugElement.injector.get(ServicesService)
        fixture.detectChanges()
        tick()
    }))

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display agent installation instruction', () => {
        // dialog should be hidden
        expect(component.displayAgentInstallationInstruction).toBeFalse()

        // prepare response for call to getMachinesServerToken
        const serverTokenResp: any = { token: 'ABC' }
        spyOn(servicesApi, 'getMachinesServerToken').and.returnValue(of(serverTokenResp))

        // show instruction
        const showBtnEl = fixture.debugElement.query(By.css('#show-agent-installation-instruction-button'))
        showBtnEl.triggerEventHandler('click', null)

        // check if it is displayed and server token retrieved
        expect(component.displayAgentInstallationInstruction).toBeTrue()
        expect(servicesApi.getMachinesServerToken).toHaveBeenCalled()
        expect(component.serverToken).toBe('ABC')

        // regenerate server token
        const regenerateMachinesServerTokenResp: any = { token: 'DEF' }
        spyOn(servicesApi, 'regenerateMachinesServerToken').and.returnValue(of(regenerateMachinesServerTokenResp))
        component.regenerateServerToken()

        // check if server token has changed
        expect(component.serverToken).toBe('DEF')

        // close instruction
        const closeBtnEl = fixture.debugElement.query(By.css('#close-agent-installation-instruction-button'))
        closeBtnEl.triggerEventHandler('click', null)

        // now dialog should be hidden
        expect(component.displayAgentInstallationInstruction).toBeFalse()
    })

    it('should list machines', fakeAsync(() => {
        // TODO: Automatic updating showUnauthorized does not work as registerOnChange is not called on selectButton
        // during component creation. fakeAsync and tick in beforeEach should help but they do not :(
        // expect(component.showUnauthorized).toBeFalse()

        // get references to select buttons
        const selectBtns = fixture.nativeElement.querySelectorAll('#unauthorized-select-button .ui-button')
        const authSelectBtnEl = selectBtns[0]
        const unauthSelectBtnEl = selectBtns[1]

        // prepare response for call to getMachines for (un)authorized machines
        const getUnauthorizedMachinesResp: any = { items: [{}, {}, {}], total: 3 }
        const getAuthorizedMachinesResp: any = { items: [{}, {}], total: 2 }
        const gmSpy = spyOn(servicesApi, 'getMachines')
        gmSpy.withArgs(0, 1, null, null, false).and.returnValue(of(getUnauthorizedMachinesResp))
        gmSpy.withArgs(0, 10, undefined, undefined, false).and.returnValue(of(getUnauthorizedMachinesResp))
        gmSpy.withArgs(0, 10, undefined, undefined, true).and.returnValue(of(getAuthorizedMachinesResp))

        // show unauthorized machines
        component.showUnauthorized = true
        unauthSelectBtnEl.dispatchEvent(new Event('click'))
        // expect(component.showUnauthorized).toBeTrue()
        expect(component.totalMachines).toBe(3)

        // show authorized machines
        component.showUnauthorized = false
        authSelectBtnEl.dispatchEvent(new Event('click'))
        // expect(component.showUnauthorized).toBeTrue()
        expect(component.totalMachines).toBe(2)
        expect(component.unauthorizedMachinesCount).toBe(3)
    }))
})
