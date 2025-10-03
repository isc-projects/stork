import { ComponentFixture, fakeAsync, flush, TestBed, tick } from '@angular/core/testing'

import { MachinesTableComponent } from './machines-table.component'
import { RouterModule } from '@angular/router'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { ButtonModule } from 'primeng/button'
import { TableHeaderCheckbox, TableModule } from 'primeng/table'
import { PanelModule } from 'primeng/panel'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { PopoverModule } from 'primeng/popover'
import { CheckboxModule } from 'primeng/checkbox'
import { FormsModule } from '@angular/forms'
import { PluralizePipe } from '../pipes/pluralize.pipe'
import { TagModule } from 'primeng/tag'
import createSpyObj = jasmine.createSpyObj
import { of, throwError } from 'rxjs'
import { AppsVersions, Machine, ServicesService } from '../backend'
import { Severity, VersionService } from '../version.service'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { TooltipModule } from 'primeng/tooltip'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { deepCopy } from '../utils'
import objectContaining = jasmine.objectContaining
import { By } from '@angular/platform-browser'
import { AppDaemonsStatusComponent } from '../app-daemons-status/app-daemons-status.component'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { ManagedAccessDirective } from '../managed-access.directive'
import { AuthService } from '../auth.service'
import { tableHasFilter } from '../table'
import { TriStateCheckboxComponent } from '../tri-state-checkbox/tri-state-checkbox.component'
import { IconFieldModule } from 'primeng/iconfield'
import { InputIconModule } from 'primeng/inputicon'

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
    const getUnauthorizedMachinesResp: any = {
        items: [
            { hostname: 'aaa', id: 1, address: 'addr1', authorized: false },
            { hostname: 'bbb', id: 2, address: 'addr2', authorized: false },
            { hostname: 'ccc', id: 3, address: 'addr3', authorized: false },
        ],
        total: 3,
    }
    const getAuthorizedMachinesResp: any = {
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
    const refreshed: Machine = {
        id: 4,
        address: 'addr zzz',
        authorized: true,
        hostname: 'new zzz',
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
                version: '2.2.0',
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
                version: '9.18.30',
            },
        ],
        agentVersion: '1.19.0',
    }

    beforeEach(async () => {
        // VersionService stub
        versionServiceStub = {
            sanitizeSemver: () => '1.2.3',
            getCurrentData: () => of({} as AppsVersions),
            getSoftwareVersionFeedback: () => ({ severity: Severity.success, messages: ['test feedback'] }),
        }

        // fake ServicesService
        servicesApi = createSpyObj('ServicesService', ['getMachines', 'getUnauthorizedMachinesCount'])

        getMachinesSpy = servicesApi.getMachines.and.returnValue(of(getAllMachinesResp))
        getMachinesSpy.withArgs(0, 10, null, null, true).and.returnValue(of(getAuthorizedMachinesResp))
        getMachinesSpy.withArgs(0, 10, null, null, false).and.returnValue(of(getUnauthorizedMachinesResp))

        await TestBed.configureTestingModule({
            declarations: [
                MachinesTableComponent,
                HelpTipComponent,
                PluralizePipe,
                VersionStatusComponent,
                LocaltimePipe,
                PlaceholderPipe,
                AppDaemonsStatusComponent,
            ],
            imports: [
                RouterModule.forRoot([]),
                ButtonModule,
                TableModule,
                PanelModule,
                BrowserAnimationsModule,
                PopoverModule,
                CheckboxModule,
                FormsModule,
                TagModule,
                TooltipModule,
                ManagedAccessDirective,
                TriStateCheckboxComponent,
                IconFieldModule,
                InputIconModule,
            ],
            providers: [
                MessageService,
                { provide: ServicesService, useValue: servicesApi },
                { provide: VersionService, useValue: versionServiceStub },
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(MachinesTableComponent)
        component = fixture.componentInstance
        msgService = fixture.debugElement.injector.get(MessageService)
        authService = fixture.debugElement.injector.get(AuthService)
        spyOn(authService, 'superAdmin').and.returnValue(true)
        fixture.detectChanges()

        // Do not save table state between tests, because that makes tests unstable.
        // spyOn(component.table, 'saveState').and.callFake(() => {})

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
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, null, null, null)
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
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, null, null, null)
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
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(100, 30, 'foo', null, true)
        expect(servicesApi.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)
        expect(component.unauthorizedMachinesCount).toBe(5)
        expect(unauthorizedMachinesCountChangeSpy).toHaveBeenCalledOnceWith(5)
        expect(component.dataLoading).toBeFalse()
    })

    xit('should apply queryParam filter value when requesting unauthorized machines data', async () => {
        // Arrange
        getMachinesSpy.and.returnValue(of(getUnauthorizedMachinesResp))

        // Act
        component.loadData({ first: 0, rows: 10, filters: {} })
        expect(component.dataLoading).toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, null, null, false)
        expect(tableHasFilter(component.machinesTable)).toBeFalse()
        // In case unauthorized machines view is loaded, unauthorized machines count is extracted from getMachines api response.
        expect(servicesApi.getUnauthorizedMachinesCount).not.toHaveBeenCalled()
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(unauthorizedMachinesCountChangeSpy).toHaveBeenCalledOnceWith(3)
        expect(component.dataLoading).toBeFalse()
        const nativeEl = fixture.nativeElement
        expect(nativeEl.textContent).toContain('aaa')
        expect(nativeEl.textContent).toContain('bbb')
        expect(nativeEl.textContent).toContain('ccc')
        expect(nativeEl.textContent).not.toContain('xxx')
        expect(nativeEl.textContent).not.toContain('zzz')
    })

    xit('should apply queryParam filter value when requesting unauthorized machines data filtered also by text', async () => {
        // Arrange
        const filter: { [k: string]: FilterMetadata } = {
            authorized: { value: null, matchMode: 'equals' },
            text: { value: 'bb', matchMode: 'contains' },
        }
        component.machinesTable.filters = filter
        const items = deepCopy(getUnauthorizedMachinesResp.items.filter((m) => m.hostname.includes('bb')))
        const response = { items: items, total: items.length }
        getMachinesSpy.and.returnValue(of(response))
        servicesApi.getUnauthorizedMachinesCount.and.returnValue(of(3))

        // Act
        component.loadData({ first: 0, rows: 10, filters: filter })
        expect(component.dataLoading).toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).toBeFalse()

        // Assert
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, 'bb', null, false)
        expect(tableHasFilter(component.machinesTable)).toBeTrue()
        expect(servicesApi.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(unauthorizedMachinesCountChangeSpy).toHaveBeenCalledOnceWith(3)

        const nativeEl = fixture.nativeElement
        expect(nativeEl.textContent).not.toContain('aaa')
        expect(nativeEl.textContent).toContain('bbb')
        expect(nativeEl.textContent).not.toContain('ccc')
        expect(nativeEl.textContent).not.toContain('xxx')
        expect(nativeEl.textContent).not.toContain('zzz')
    })

    xit('should apply queryParam filter value when requesting authorized machines data', async () => {
        // Arrange
        getMachinesSpy.and.returnValue(of(getAuthorizedMachinesResp))
        servicesApi.getUnauthorizedMachinesCount.and.returnValue(of(3))

        // Act
        component.loadData({ first: 0, rows: 10, filters: {} })
        expect(component.dataLoading).toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, null, null, true)
        expect(tableHasFilter(component.machinesTable)).toBeFalse()
        expect(servicesApi.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(unauthorizedMachinesCountChangeSpy).toHaveBeenCalledOnceWith(3)
        expect(component.dataLoading).toBeFalse()
        const nativeEl = fixture.nativeElement
        expect(nativeEl.textContent).not.toContain('aaa')
        expect(nativeEl.textContent).not.toContain('bbb')
        expect(nativeEl.textContent).not.toContain('ccc')
        expect(nativeEl.textContent).toContain('xxx')
        expect(nativeEl.textContent).toContain('zzz')
    })

    xit('should respect queryParam filter value when table was filtered by other value', async () => {
        // Arrange
        const filter: { [k: string]: FilterMetadata } = {
            authorized: { value: true, matchMode: 'equals' },
            text: { value: null, matchMode: 'contains' },
        }
        component.machinesTable.filters = filter
        getMachinesSpy.and.returnValue(of(getUnauthorizedMachinesResp))

        // Act
        component.loadData({ first: 0, rows: 10, filters: filter })
        expect(component.dataLoading).toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, null, null, false)
        expect(tableHasFilter(component.machinesTable)).toBeFalse()
        expect(servicesApi.getUnauthorizedMachinesCount).not.toHaveBeenCalled()
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(unauthorizedMachinesCountChangeSpy).toHaveBeenCalledOnceWith(3)
        expect(component.dataLoading).toBeFalse()
        const nativeEl = fixture.nativeElement
        expect(nativeEl.textContent).toContain('aaa')
        expect(nativeEl.textContent).toContain('bbb')
        expect(nativeEl.textContent).toContain('ccc')
        expect(nativeEl.textContent).not.toContain('xxx')
        expect(nativeEl.textContent).not.toContain('zzz')
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
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, null, null, null)
        expect(component.dataCollection).toEqual([])
        expect(component.totalRecords).toBeUndefined()
        expect(msgSpy).toHaveBeenCalledOnceWith(
            objectContaining({ severity: 'error', summary: 'Cannot get machine list' })
        )
        expect(servicesApi.getUnauthorizedMachinesCount).not.toHaveBeenCalled()
        expect(component.dataLoading).toBeFalse()
    })

    it('should return authorized machines displayed', () => {
        // Arrange
        component.dataCollection = getAllMachinesResp.items

        // Act & Assert
        expect(component.authorizedMachinesDisplayed()).toBeTrue()
    })

    it('should not return authorized machines displayed', () => {
        // Arrange
        component.dataCollection = getUnauthorizedMachinesResp.items

        // Act & Assert
        expect(component.authorizedMachinesDisplayed()).toBeFalse()
    })

    it('should return unauthorized machines displayed', () => {
        // Arrange
        component.dataCollection = getAllMachinesResp.items

        // Act & Assert
        expect(component.unauthorizedMachinesDisplayed()).toBeTrue()
    })

    it('should not return unauthorized machines displayed', () => {
        // Arrange
        component.dataCollection = getAuthorizedMachinesResp.items

        // Act & Assert
        expect(component.unauthorizedMachinesDisplayed()).toBeFalse()
    })

    xit('should delete a machine from data collection', () => {
        // Arrange
        component.dataCollection = deepCopy(getUnauthorizedMachinesResp.items)

        // Act
        // component.deleteMachine(1)

        // Assert
        expect(component.dataCollection.length).toBe(2)
        expect(component.dataCollection).toContain({ hostname: 'bbb', id: 2, address: 'addr2', authorized: false })
        expect(component.dataCollection).toContain({ hostname: 'ccc', id: 3, address: 'addr3', authorized: false })
        expect(servicesApi.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)
    })

    xit('should not delete a machine from data collection', () => {
        // Arrange
        component.dataCollection = getUnauthorizedMachinesResp.items

        // Act
        // component.deleteMachine(4)

        // Assert
        expect(component.dataCollection.length).toBe(3)
        expect(component.dataCollection).toBe(getUnauthorizedMachinesResp.items)
        expect(servicesApi.getUnauthorizedMachinesCount).not.toHaveBeenCalled()
    })

    xit('should not fail when trying to delete a machine when data collection is undefined', () => {
        // Arrange & Act & Assert
        // component.deleteMachine(4)
        expect(component.dataCollection).toBeFalsy()
        expect(servicesApi.getUnauthorizedMachinesCount).not.toHaveBeenCalled()
    })

    xit('should refresh machine state', () => {
        // Arrange
        component.dataCollection = deepCopy(getAuthorizedMachinesResp.items)

        // Act
        // component.refreshMachineState(refreshed)

        // Assert
        const changedMachine = component.dataCollection.find((m) => m.id === 4)
        expect(changedMachine).toBeTruthy()
        expect(changedMachine).toEqual(refreshed)
    })

    xit('should not refresh machine state', () => {
        // Arrange
        component.dataCollection = getUnauthorizedMachinesResp.items

        // Act
        // component.refreshMachineState(refreshed)

        // Assert
        const changedMachine = component.dataCollection.find((m) => m.id === 4)
        expect(changedMachine).toBeUndefined()
        expect(component.dataCollection).toEqual(getUnauthorizedMachinesResp.items)
    })

    xit('should not fail when trying to refresh a machine when data collection is undefined', () => {
        // Arrange & Act & Assert
        // component.refreshMachineState(refreshed)
        expect(component.dataCollection).toBeFalsy()
    })

    it('should display status of all daemons from all applications', async () => {
        // Arrange
        const oneMachineResponse = {
            items: [{ hostname: 'zzz', id: 4, authorized: true }],
            total: 1,
        }
        getMachinesSpy.and.returnValue(of(oneMachineResponse))
        component.loadData({ first: 0, rows: 10, filters: {} })
        expect(component.dataLoading).withContext('data is loading').toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).withContext('data loading done').toBeFalse()

        // Act
        // component.refreshMachineState(refreshed)
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
        expect(versionStatus.length).toEqual(3)

        // Check if versions and apps match.
        expect(versionStatus[0].properties.outerHTML).toContain('1.19.0')
        expect(versionStatus[0].properties.outerHTML).toContain('stork')

        expect(versionStatus[1].properties.outerHTML).toContain('2.2.0')
        expect(versionStatus[1].properties.outerHTML).toContain('kea')

        expect(versionStatus[2].properties.outerHTML).toContain('9.18.30')
        expect(versionStatus[2].properties.outerHTML).toContain('bind9')

        // All VersionStatus components got Severity.success and 'test feedback' message from Version Service stub
        expect(versionStatus[0].properties.outerHTML).toContain('text-green-500')
        expect(versionStatus[0].properties.outerHTML).toContain('test feedback')
        expect(versionStatus[1].properties.outerHTML).toContain('text-green-500')
        expect(versionStatus[1].properties.outerHTML).toContain('test feedback')
        expect(versionStatus[2].properties.outerHTML).toContain('text-green-500')
        expect(versionStatus[2].properties.outerHTML).toContain('test feedback')
    })

    it('should set data loading state', () => {
        // Arrange & Act & Assert
        component.setDataLoading(true)
        expect(component.dataLoading).toBeTrue()
        component.setDataLoading(false)
        expect(component.dataLoading).toBeFalse()
    })

    xit('should clear selected machines', fakeAsync(() => {
        // Arrange
        component.loadData({ first: 0, rows: 10, filters: {} })
        expect(component.dataLoading).withContext('data is loading').toBeTrue()
        tick()
        fixture.detectChanges()
        expect(component.dataLoading).withContext('data loading done').toBeFalse()

        const checkboxes = fixture.debugElement.queryAll(By.css('table .p-checkbox:not(.p-disabled)'))
        expect(checkboxes).toBeTruthy()
        expect(checkboxes.length)
            .withContext('there should be 1 "select all" checkbox and 3 checkboxes for each unauthorized machine')
            .toEqual(4)
        const selectAllCheckbox = checkboxes[0]
        expect(selectAllCheckbox).toBeTruthy()

        selectAllCheckbox.nativeElement.click()
        fixture.detectChanges()
        fixture.detectChanges() // PrimeNG TableHeaderCheckbox has complicated chain of change detection, so call detectChanges additionally.
        flush() // Wait for PrimeNG to react on Select all change

        expect(component.selectedMachines.length).toEqual(3)
        expect(component.selectedMachines).toEqual(getUnauthorizedMachinesResp.items)
        for (const ch of checkboxes) {
            const component = ch.componentInstance as TableHeaderCheckbox
            expect(component).not.toBeNull()
            expect(component.checked).withContext('checkbox should be selected').toBeTrue()
        }

        // Act
        component.clearSelection()
        tick()
        fixture.detectChanges()
        tick()

        // Assert
        expect(component.selectedMachines.length).toEqual(0)
        for (const ch of checkboxes) {
            const component = ch.componentInstance as TableHeaderCheckbox
            expect(component).not.toBeNull()
            expect(component.checked).withContext('checkbox should not be selected').toBeFalse()
        }
    }))

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

        const checkboxes = fixture.debugElement.queryAll(By.css('table .p-checkbox'))
        expect(checkboxes).toBeTruthy()
        expect(checkboxes.length)
            .withContext('there should be 1 "select all" checkbox and 5 checkboxes for each machine')
            .toEqual(6)
        const selectAllCheckbox = checkboxes[0]
        expect(selectAllCheckbox).toBeTruthy()

        selectAllCheckbox.nativeElement.click()
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

    xit('should select or deselect only unauthorized machines', async () => {
        // Arrange
        component.loadData({ first: 0, rows: 10, filters: {} })
        expect(component.dataLoading).withContext('data is loading').toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).withContext('data loading done').toBeFalse()

        const checkboxes = fixture.debugElement.queryAll(By.css('table .p-checkbox:not(.p-disabled)'))
        expect(checkboxes).toBeTruthy()
        expect(checkboxes.length)
            .withContext('there should be 1 "select all" checkbox and 3 checkboxes for each unauthorized machine')
            .toEqual(4)
        const selectAllCheckbox = checkboxes[0]
        expect(selectAllCheckbox).toBeTruthy()

        const disabledCheckboxes = fixture.debugElement.queryAll(By.css('table .p-checkbox.p-disabled'))
        expect(disabledCheckboxes).toBeTruthy()
        expect(disabledCheckboxes.length)
            .withContext('there should be 2 disabled checkboxes for authorized machines')
            .toEqual(2)

        // Act & Assert
        selectAllCheckbox.nativeElement.click() // select All unauthorized
        await fixture.whenStable()
        fixture.detectChanges()
        await fixture.whenStable() // Wait for PrimeNG to react on Select all change
        fixture.detectChanges()

        expect(component.selectedMachines.length).toEqual(3)
        expect(component.selectedMachines).toEqual(getUnauthorizedMachinesResp.items)
        for (const ch of checkboxes) {
            const box = ch.query(By.css('.p-checkbox-box'))
            expect(box).toBeTruthy()
            expect(box.nativeElement).withContext('checkbox should be selected').toHaveClass('p-highlight')
        }

        selectAllCheckbox.nativeElement.click() // deselect All unauthorized
        await fixture.whenStable()
        fixture.detectChanges()
        await fixture.whenStable() // Wait for PrimeNG to react on Select all change
        fixture.detectChanges()

        expect(component.selectedMachines.length).toEqual(0)
        for (const ch of checkboxes) {
            const box = ch.query(By.css('.p-checkbox-box'))
            expect(box).toBeTruthy()
            expect(box.nativeElement).withContext('checkbox should not be selected').not.toHaveClass('p-highlight')
        }

        const selectAllBox = selectAllCheckbox.query(By.css('.p-checkbox-box'))
        expect(selectAllBox).toBeTruthy()
        expect(selectAllBox.nativeElement).withContext('checkbox should not be selected').not.toHaveClass('p-highlight')

        // Click checkboxes one by one.
        checkboxes[1].nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()
        checkboxes[2].nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()
        checkboxes[3].nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()

        expect(component.selectedMachines.length).toEqual(3)
        expect(component.selectedMachines).toEqual(getUnauthorizedMachinesResp.items)
        expect(selectAllBox.nativeElement).withContext('checkbox should be selected').toHaveClass('p-highlight')

        // Deselect one machine.
        checkboxes[3].nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()

        expect(component.selectedMachines.length).toEqual(2)
        expect(selectAllBox.nativeElement).withContext('checkbox should not be selected').not.toHaveClass('p-highlight')
    })
})
