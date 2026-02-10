import { ComponentFixture, TestBed } from '@angular/core/testing'

import { MachinesTableComponent } from './machines-table.component'
import { provideRouter } from '@angular/router'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { provideAnimations } from '@angular/platform-browser/animations'
import createSpyObj = jasmine.createSpyObj
import { of, throwError } from 'rxjs'
import { AppsVersions, Machines, ServicesService } from '../backend'
import { Severity, VersionService } from '../version.service'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import objectContaining = jasmine.objectContaining
import { By } from '@angular/platform-browser'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { AuthService } from '../auth.service'

describe('MachinesTableComponent', () => {
    let component: MachinesTableComponent
    let fixture: ComponentFixture<MachinesTableComponent>
    let versionServiceStub: Partial<VersionService>
    let servicesApi: any
    let getMachinesSpy: any
    let unauthorizedMachinesCountChangeSpy: any
    let msgService: MessageService
    let authService: AuthService

    // prepare responses for api calls
    const getUnauthorizedMachinesResp: Machines = {
        items: [
            { hostname: 'aaa', id: 1, address: 'addr1', authorized: false },
            { hostname: 'bbb', id: 2, address: 'addr2', authorized: false },
            { hostname: 'ccc', id: 3, address: 'addr3', authorized: false },
        ],
        total: 3,
    }
    const getAuthorizedMachinesResp: Machines = {
        items: [
            { hostname: 'zzz', id: 4, address: 'addr4', authorized: true },
            { hostname: 'xxx', id: 5, address: 'addr5', authorized: true },
        ],
        total: 2,
    }
    const getAllMachinesResp = {
        items: [...getUnauthorizedMachinesResp.items, ...getAuthorizedMachinesResp.items],
        total: 5,
    }

    beforeEach(async () => {
        // VersionService stub
        versionServiceStub = {
            sanitizeSemver: (version: string) => version,
            getCurrentData: () => of({} as AppsVersions),
            getSoftwareVersionFeedback: () => ({ severity: Severity.success, messages: ['test feedback'] }),
        }

        // fake ServicesService
        servicesApi = createSpyObj('ServicesService', ['getMachines', 'getUnauthorizedMachinesCount'])

        getMachinesSpy = servicesApi.getMachines.and.returnValue(of(getAllMachinesResp))
        getMachinesSpy.withArgs(0, 10, null, null, true).and.returnValue(of(getAuthorizedMachinesResp))
        getMachinesSpy.withArgs(0, 10, null, null, false).and.returnValue(of(getUnauthorizedMachinesResp))

        await TestBed.configureTestingModule({
            providers: [
                MessageService,
                { provide: ServicesService, useValue: servicesApi },
                { provide: VersionService, useValue: versionServiceStub },
                provideAnimations(),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideRouter([]),
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(MachinesTableComponent)
        component = fixture.componentInstance
        msgService = fixture.debugElement.injector.get(MessageService)
        authService = fixture.debugElement.injector.get(AuthService)
        spyOn(authService, 'superAdmin').and.returnValue(true)
        fixture.detectChanges()

        unauthorizedMachinesCountChangeSpy = spyOn(component.unauthorizedMachinesCountChange, 'emit')
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should load all machines', async () => {
        // Arrange
        servicesApi.getUnauthorizedMachinesCount.and.returnValue(of(3))

        // Act
        component.loadData({ first: 0, rows: 10, filters: {} })
        expect(component.dataLoading).toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, null, null, null, null)
        expect(component.dataCollection).toBe(getAllMachinesResp.items)
        expect(component.totalRecords).toBe(5)
        expect(servicesApi.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(unauthorizedMachinesCountChangeSpy).toHaveBeenCalledOnceWith(3)
        expect(component.dataLoading).toBeFalse()
        const nativeEl = fixture.nativeElement
        expect(nativeEl.textContent).toContain('aaa')
        expect(nativeEl.textContent).toContain('bbb')
        expect(nativeEl.textContent).toContain('ccc')
        expect(nativeEl.textContent).toContain('xxx')
        expect(nativeEl.textContent).toContain('zzz')
    })

    it('should not load malformed data', async () => {
        // Arrange
        getMachinesSpy.and.returnValue(of({ foo: 'bar', count: 123 }))

        // Act
        component.loadData({ first: 0, rows: 10, filters: {} })
        expect(component.dataLoading).toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, null, null, null, null)
        expect(component.dataCollection).toEqual([])
        expect(component.totalRecords).toBe(0)
        expect(component.dataLoading).toBeFalse()
    })

    it('should apply filters when requesting data', async () => {
        // Arrange
        const filter: { [k: string]: FilterMetadata } = {
            authorized: { value: true, matchMode: 'equals' },
            text: { value: 'foo', matchMode: 'contains' },
        }
        servicesApi.getUnauthorizedMachinesCount.and.returnValue(of(5))

        // Act
        component.loadData({ first: 100, rows: 30, filters: filter })
        expect(component.dataLoading).toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(100, 30, 'foo', true, null, null)
        expect(servicesApi.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)
        expect(component.unauthorizedMachinesCount).toBe(5)
        expect(unauthorizedMachinesCountChangeSpy).toHaveBeenCalledOnceWith(5)
        expect(component.dataLoading).toBeFalse()
    })

    it('should display error when api call fails', async () => {
        // Arrange
        const msgSpy = spyOn(msgService, 'add')
        getMachinesSpy.and.returnValue(throwError(() => new Error('test error')))

        // Act
        component.loadData({ first: 0, rows: 10, filters: {} })
        expect(component.dataLoading).toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, null, null, null, null)
        expect(component.dataCollection).toEqual([])
        expect(component.totalRecords).toBeUndefined()
        expect(msgSpy).toHaveBeenCalledOnceWith(
            objectContaining({ severity: 'error', summary: 'Cannot get machine list' })
        )
        expect(servicesApi.getUnauthorizedMachinesCount).not.toHaveBeenCalled()
        expect(component.dataLoading).toBeFalse()
    })

    it('should return authorized machines displayed', async () => {
        // Arrange
        component.loadData({ first: 0, rows: 10, filters: {} })

        // Act & Assert
        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.authorizedMachinesDisplayed()).toBeTrue()
    })

    it('should not return authorized machines displayed', () => {
        // Arrange
        component.dataCollection = getUnauthorizedMachinesResp.items

        // Act & Assert
        expect(component.authorizedMachinesDisplayed()).toBeFalse()
    })

    it('should return unauthorized machines displayed', async () => {
        // Arrange
        component.loadData({ first: 0, rows: 10, filters: {} })

        // Act & Assert
        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.unauthorizedMachinesDisplayed()).toBeTrue()
    })

    it('should not return unauthorized machines displayed', () => {
        // Arrange
        component.dataCollection = getAuthorizedMachinesResp.items

        // Act & Assert
        expect(component.unauthorizedMachinesDisplayed()).toBeFalse()
    })

    it('should display status of all daemons from all daemons', async () => {
        // Prepare the data
        const getMachinesResp: Machines = {
            items: [
                {
                    id: 1,
                    authorized: true,
                    hostname: 'zzz',
                    address: 'fe80::1',
                    daemons: [
                        {
                            active: true,
                            extendedVersion: '2.2.0',
                            id: 1,
                            name: 'dhcp4',
                            label: 'DHCPv4@myhost.example.org',
                            version: '2.2.0',
                        },
                        {
                            active: false,
                            extendedVersion: '2.3.0',
                            id: 2,
                            name: 'ca',
                            label: 'CA@myhost.example.org',
                            version: '2.2.0',
                        },
                        {
                            active: true,
                            id: 3,
                            name: 'named',
                            label: 'named@myhost.example.org',
                            version: '9.18.30',
                        },
                    ],
                    agentVersion: '1.19.0',
                },
            ],
            total: 1,
        }

        // Arrange
        getMachinesSpy.and.returnValue(of(getMachinesResp))
        component.loadData({ first: 0, rows: 10, filters: {} })
        expect(component.dataLoading).withContext('data is loading').toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).withContext('data loading done').toBeFalse()

        // Act
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        const textContent = fixture.nativeElement.innerText

        expect(textContent).toContain('DHCPv4')
        expect(textContent).toContain('CA')
        expect(textContent).toContain('named')

        // One VersionStatus for Stork agent + one for Kea + one for BIND9.
        const versionStatus = fixture.debugElement.queryAll(By.directive(VersionStatusComponent))
        expect(versionStatus).toBeTruthy()
        expect(versionStatus.length).toEqual(4) // 3 daemons + Stork agent

        // Check if versions and apps match.
        expect(versionStatus[0].nativeElement.innerHTML).toContain('1.19.0')
        expect(versionStatus[0].nativeElement.innerHTML).not.toContain('Stork')

        expect(versionStatus[1].nativeElement.innerHTML).toContain('2.2.0')
        expect(versionStatus[1].nativeElement.innerHTML).not.toContain('DHCPv4')

        expect(versionStatus[2].nativeElement.innerHTML).toContain('2.2.0')
        expect(versionStatus[2].nativeElement.innerHTML).not.toContain('CA')

        expect(versionStatus[3].nativeElement.innerHTML).toContain('9.18.30')
        expect(versionStatus[3].nativeElement.innerHTML).not.toContain('named')

        // All VersionStatus components got Severity.success and 'test feedback' message from Version Service stub
        expect(versionStatus[0].nativeElement.innerHTML).toContain('text-green-500')
        expect(versionStatus[0].nativeElement.innerHTML).toContain('test feedback')
        expect(versionStatus[1].nativeElement.innerHTML).toContain('text-green-500')
        expect(versionStatus[1].nativeElement.innerHTML).toContain('test feedback')
        expect(versionStatus[2].nativeElement.innerHTML).toContain('text-green-500')
        expect(versionStatus[2].nativeElement.innerHTML).toContain('test feedback')
    })

    it('should set data loading state', () => {
        // Arrange & Act & Assert
        component.setDataLoading(true)
        expect(component.dataLoading).toBeTrue()
        component.setDataLoading(false)
        expect(component.dataLoading).toBeFalse()
    })

    it('should clear selected machines', () => {
        // Arrange
        component.selectedMachines.set(getUnauthorizedMachinesResp.items)
        expect(component.selectedMachines().length).toEqual(getUnauthorizedMachinesResp.items.length)

        // Act
        component.clearSelection()

        // Assert
        expect(component.selectedMachines().length).toEqual(0)
    })

    it('should emit on machine menu display', async () => {
        // Arrange
        const eventEmitterSpy = spyOn(component.machineMenuDisplay, 'emit')
        component.loadData({ first: 0, rows: 10, filters: {} })
        expect(component.dataLoading).withContext('data is loading').toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).withContext('data loading done').toBeFalse()

        const buttonDe = fixture.debugElement.query(By.css('#show-machines-menu-1'))
        expect(buttonDe).toBeTruthy()

        const machine = getAllMachinesResp.items.find((m) => m.id === 1)
        expect(machine).toBeTruthy()

        // Act
        buttonDe.nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(eventEmitterSpy).toHaveBeenCalledOnceWith(objectContaining({ e: jasmine.any(Event), m: machine }))
    })

    it('should emit on authorize selected machines', async () => {
        // Arrange
        const eventEmitterSpy = spyOn(component.authorizeSelectedMachines, 'emit')
        component.loadData({ first: 0, rows: 10, filters: {} })
        expect(component.dataLoading).withContext('data is loading').toBeTrue()

        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).withContext('data loading done').toBeFalse()
        expect(component.dataCollection.length).toBeGreaterThan(0)

        const checkboxes = fixture.debugElement.queryAll(By.css('table input[type="checkbox"]'))
        expect(checkboxes).toBeTruthy()
        expect(checkboxes.length)
            .withContext('there should be 1 "select all" checkbox and 5 checkboxes for each machine')
            .toEqual(6)
        const selectAllCheckbox = checkboxes[0]
        expect(selectAllCheckbox).toBeTruthy()

        selectAllCheckbox.nativeElement.checked = true
        selectAllCheckbox.nativeElement.dispatchEvent(new Event('change'))
        await fixture.whenStable()
        fixture.detectChanges()
        fixture.detectChanges() // PrimeNG TableHeaderCheckbox has complicated chain of change detection, so call detectChanges additionally.

        const authorizeBtnDe = fixture.debugElement.query(By.css('#authorize-selected-button button'))
        expect(authorizeBtnDe).toBeTruthy()
        expect(authorizeBtnDe.nativeElement).not.toHaveClass('p-disabled')

        // Act
        authorizeBtnDe.nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(eventEmitterSpy).toHaveBeenCalledOnceWith(getUnauthorizedMachinesResp.items)
    })

    it('should have enabled or disabled button in filtering toolbar according to privileges', () => {
        expect(component.toolbarButtons.length).toBeGreaterThan(0)
        // at first, it should be disabled
        expect(component.toolbarButtons[0].disabled).toBeTrue()
        // it should react on signals change
        component.canAuthorizeMachine.set(true)
        component.unauthorizedMachinesDisplayed.set(true)
        component.selectedMachines.set([{ id: 1, address: 'abc' }])
        fixture.detectChanges()
        expect(component.toolbarButtons[0].disabled).toBeFalse()
    })
})
