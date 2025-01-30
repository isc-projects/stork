import { ComponentFixture, TestBed } from '@angular/core/testing'

import { MachinesTableComponent } from './machines-table.component'
import { RouterModule } from '@angular/router'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { ButtonModule } from 'primeng/button'
import { TableModule } from 'primeng/table'
import { PanelModule } from 'primeng/panel'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TriStateCheckboxModule } from 'primeng/tristatecheckbox'
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

describe('MachinesTableComponent', () => {
    let component: MachinesTableComponent
    let fixture: ComponentFixture<MachinesTableComponent>
    let versionServiceStub: Partial<VersionService>
    let servicesApi: any
    let getMachinesSpy: any
    let unauthorizedMachinesCountChangeSpy: any
    let msgService: MessageService

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
            imports: [
                RouterModule.forRoot([]),
                HttpClientTestingModule,
                ButtonModule,
                TableModule,
                PanelModule,
                BrowserAnimationsModule,
                OverlayPanelModule,
                TriStateCheckboxModule,
                FormsModule,
                TagModule,
                TooltipModule,
            ],
            declarations: [
                MachinesTableComponent,
                HelpTipComponent,
                PluralizePipe,
                VersionStatusComponent,
                LocaltimePipe,
                PlaceholderPipe,
                AppDaemonsStatusComponent,
            ],
            providers: [
                MessageService,
                { provide: ServicesService, useValue: servicesApi },
                { provide: VersionService, useValue: versionServiceStub },
            ],
        }).compileComponents()

        fixture = TestBed.createComponent(MachinesTableComponent)
        component = fixture.componentInstance
        msgService = fixture.debugElement.injector.get(MessageService)
        fixture.detectChanges()

        // Do not save table state between tests, because that makes tests unstable.
        spyOn(component.table, 'saveState').and.callFake(() => {})

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
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, null, null, null)
        expect(component.dataCollection).toBe(getAllMachinesResp.items)
        expect(component.totalRecords).toBe(5)
        expect(servicesApi.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(unauthorizedMachinesCountChangeSpy).toHaveBeenCalledOnceWith(3)
        expect(component.dataLoading).toBeTrue()

        await fixture.whenStable()
        fixture.detectChanges()
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
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, null, null, null)
        expect(component.dataCollection).toEqual([])
        expect(component.totalRecords).toBe(0)
        expect(component.dataLoading).toBeTrue()

        await fixture.whenStable()
        fixture.detectChanges()
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
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(component.prefilterValue).toBeNull()
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(100, 30, 'foo', null, true)
        expect(servicesApi.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)
        expect(component.unauthorizedMachinesCount).toBe(5)
        expect(unauthorizedMachinesCountChangeSpy).toHaveBeenCalledOnceWith(5)
        expect(component.dataLoading).toBeTrue()

        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).toBeFalse()
    })

    it('should apply queryParam filter value when requesting unauthorized machines data', async () => {
        // Arrange
        component.prefilterValue = false
        getMachinesSpy.and.returnValue(of(getUnauthorizedMachinesResp))

        // Act
        component.loadData({ first: 0, rows: 10, filters: {} })
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, null, null, false)
        expect(component.hasFilter(component.table)).toBeFalse()
        // In case unauthorized machines view is loaded, unauthorized machines count is extracted from getMachines api response.
        expect(servicesApi.getUnauthorizedMachinesCount).not.toHaveBeenCalled()
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(unauthorizedMachinesCountChangeSpy).toHaveBeenCalledOnceWith(3)
        expect(component.dataLoading).toBeTrue()

        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).toBeFalse()
        const nativeEl = fixture.nativeElement
        expect(nativeEl.textContent).toContain('aaa')
        expect(nativeEl.textContent).toContain('bbb')
        expect(nativeEl.textContent).toContain('ccc')
        expect(nativeEl.textContent).not.toContain('xxx')
        expect(nativeEl.textContent).not.toContain('zzz')
    })

    it('should apply queryParam filter value when requesting unauthorized machines data filtered also by text', async () => {
        // Arrange
        component.prefilterValue = false
        const filter: { [k: string]: FilterMetadata } = {
            authorized: { value: null, matchMode: 'equals' },
            text: { value: 'bb', matchMode: 'contains' },
        }
        component.table.filters = filter
        const items = deepCopy(getUnauthorizedMachinesResp.items.filter((m) => m.hostname.includes('bb')))
        const response = { items: items, total: items.length }
        getMachinesSpy.and.returnValue(of(response))
        servicesApi.getUnauthorizedMachinesCount.and.returnValue(of(3))

        // Act
        component.loadData({ first: 0, rows: 10, filters: filter })
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, 'bb', null, false)
        expect(component.hasFilter(component.table)).toBeTrue()
        expect(servicesApi.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(unauthorizedMachinesCountChangeSpy).toHaveBeenCalledOnceWith(3)
        expect(component.dataLoading).toBeTrue()

        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).toBeFalse()
        const nativeEl = fixture.nativeElement
        expect(nativeEl.textContent).not.toContain('aaa')
        expect(nativeEl.textContent).toContain('bbb')
        expect(nativeEl.textContent).not.toContain('ccc')
        expect(nativeEl.textContent).not.toContain('xxx')
        expect(nativeEl.textContent).not.toContain('zzz')
    })

    it('should apply queryParam filter value when requesting authorized machines data', async () => {
        // Arrange
        component.prefilterValue = true
        getMachinesSpy.and.returnValue(of(getAuthorizedMachinesResp))
        servicesApi.getUnauthorizedMachinesCount.and.returnValue(of(3))

        // Act
        component.loadData({ first: 0, rows: 10, filters: {} })
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, null, null, true)
        expect(component.hasFilter(component.table)).toBeFalse()
        expect(servicesApi.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(unauthorizedMachinesCountChangeSpy).toHaveBeenCalledOnceWith(3)
        expect(component.dataLoading).toBeTrue()

        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).toBeFalse()
        const nativeEl = fixture.nativeElement
        expect(nativeEl.textContent).not.toContain('aaa')
        expect(nativeEl.textContent).not.toContain('bbb')
        expect(nativeEl.textContent).not.toContain('ccc')
        expect(nativeEl.textContent).toContain('xxx')
        expect(nativeEl.textContent).toContain('zzz')
    })

    it('should respect queryParam filter value when table was filtered by other value', async () => {
        // Arrange
        component.prefilterValue = false
        const filter: { [k: string]: FilterMetadata } = {
            authorized: { value: true, matchMode: 'equals' },
            text: { value: null, matchMode: 'contains' },
        }
        component.table.filters = filter
        getMachinesSpy.and.returnValue(of(getUnauthorizedMachinesResp))

        // Act
        component.loadData({ first: 0, rows: 10, filters: filter })
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, null, null, false)
        expect(component.hasFilter(component.table)).toBeFalse()
        expect(servicesApi.getUnauthorizedMachinesCount).not.toHaveBeenCalled()
        expect(component.unauthorizedMachinesCount).toBe(3)
        expect(unauthorizedMachinesCountChangeSpy).toHaveBeenCalledOnceWith(3)
        expect(component.dataLoading).toBeTrue()

        await fixture.whenStable()
        fixture.detectChanges()
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
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, null, null, null)
        expect(component.dataCollection).toBeFalsy()
        expect(component.totalRecords).toBe(0)
        expect(msgSpy).toHaveBeenCalledOnceWith(
            objectContaining({ severity: 'error', summary: 'Cannot get machine list' })
        )
        expect(servicesApi.getUnauthorizedMachinesCount).not.toHaveBeenCalled()
        expect(component.dataLoading).toBeTrue()

        await fixture.whenStable()
        fixture.detectChanges()
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

    it('should delete a machine from data collection', () => {
        // Arrange
        component.dataCollection = deepCopy(getUnauthorizedMachinesResp.items)

        // Act
        component.deleteMachine(1)

        // Assert
        expect(component.dataCollection.length).toBe(2)
        expect(component.dataCollection).toContain({ hostname: 'bbb', id: 2, address: 'addr2', authorized: false })
        expect(component.dataCollection).toContain({ hostname: 'ccc', id: 3, address: 'addr3', authorized: false })
        expect(servicesApi.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)
    })

    it('should not delete a machine from data collection', () => {
        // Arrange
        component.dataCollection = getUnauthorizedMachinesResp.items

        // Act
        component.deleteMachine(4)

        // Assert
        expect(component.dataCollection.length).toBe(3)
        expect(component.dataCollection).toBe(getUnauthorizedMachinesResp.items)
        expect(servicesApi.getUnauthorizedMachinesCount).not.toHaveBeenCalled()
    })

    it('should not fail when trying to delete a machine when data collection is undefined', () => {
        // Arrange & Act & Assert
        component.deleteMachine(4)
        expect(component.dataCollection).toBeFalsy()
        expect(servicesApi.getUnauthorizedMachinesCount).not.toHaveBeenCalled()
    })

    it('should refresh machine state', () => {
        // Arrange
        component.dataCollection = deepCopy(getAuthorizedMachinesResp.items)

        // Act
        component.refreshMachineState(refreshed)

        // Assert
        const changedMachine = component.dataCollection.find((m) => m.id === 4)
        expect(changedMachine).toBeTruthy()
        expect(changedMachine).toEqual(refreshed)
    })

    it('should not refresh machine state', () => {
        // Arrange
        component.dataCollection = getUnauthorizedMachinesResp.items

        // Act
        component.refreshMachineState(refreshed)

        // Assert
        const changedMachine = component.dataCollection.find((m) => m.id === 4)
        expect(changedMachine).toBeUndefined()
        expect(component.dataCollection).toEqual(getUnauthorizedMachinesResp.items)
    })

    it('should not fail when trying to refresh a machine when data collection is undefined', () => {
        // Arrange & Act & Assert
        component.refreshMachineState(refreshed)
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
        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).withContext('data is loading').toBeTrue()

        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).withContext('data loading done').toBeFalse()

        // Act
        component.refreshMachineState(refreshed)
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

    it('should clear selected machines', async () => {
        // Arrange
        component.loadData({ first: 0, rows: 10, filters: {} })
        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).withContext('data is loading').toBeTrue()

        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).withContext('data loading done').toBeFalse()

        const checkboxes = fixture.debugElement.queryAll(By.css('table .p-checkbox .p-checkbox-box:not(.p-disabled)'))
        expect(checkboxes).toBeTruthy()
        expect(checkboxes.length)
            .withContext('there should be 1 "select all" checkbox and 3 checkboxes for each unauthorized machine')
            .toEqual(4)
        const selectAllCheckbox = checkboxes[0]
        expect(selectAllCheckbox).toBeTruthy()

        selectAllCheckbox.nativeElement.click()

        await fixture.whenStable()
        fixture.detectChanges()
        await fixture.whenStable() // Wait for PrimeNG to react on Select all change
        fixture.detectChanges()

        expect(component.selectedMachines.length).toEqual(3)
        expect(component.selectedMachines).toEqual(getUnauthorizedMachinesResp.items)
        for (const ch of checkboxes) {
            expect(ch.nativeElement).withContext('checkbox should be selected').toHaveClass('p-highlight')
        }

        // Act
        component.clearSelection()
        await fixture.whenStable()
        fixture.detectChanges()
        await fixture.whenStable() // Wait for PrimeNG to react on Select all change
        fixture.detectChanges()

        // Assert
        expect(component.selectedMachines.length).toEqual(0)
        for (const ch of checkboxes) {
            expect(ch.nativeElement).withContext('checkbox should not be selected').not.toHaveClass('p-highlight')
        }
    })

    it('should emit on machine menu display', async () => {
        // Arrange
        const eventEmitterSpy = spyOn(component.machineMenuDisplay, 'emit')
        component.loadData({ first: 0, rows: 10, filters: {} })
        await fixture.whenStable()
        fixture.detectChanges()
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
        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).withContext('data is loading').toBeTrue()

        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).withContext('data loading done').toBeFalse()

        const checkboxes = fixture.debugElement.queryAll(By.css('table .p-checkbox .p-checkbox-box'))
        expect(checkboxes).toBeTruthy()
        expect(checkboxes.length)
            .withContext('there should be 1 "select all" checkbox and 5 checkboxes for each machine')
            .toEqual(6)
        const selectAllCheckbox = checkboxes[0]
        expect(selectAllCheckbox).toBeTruthy()

        selectAllCheckbox.nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()

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

    it('should select or deselect only unauthorized machines', async () => {
        // Arrange
        component.loadData({ first: 0, rows: 10, filters: {} })
        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).withContext('data is loading').toBeTrue()

        await fixture.whenStable()
        fixture.detectChanges()
        expect(component.dataLoading).withContext('data loading done').toBeFalse()

        const checkboxes = fixture.debugElement.queryAll(By.css('table .p-checkbox .p-checkbox-box:not(.p-disabled)'))
        expect(checkboxes).toBeTruthy()
        expect(checkboxes.length)
            .withContext('there should be 1 "select all" checkbox and 3 checkboxes for each unauthorized machine')
            .toEqual(4)
        const selectAllCheckbox = checkboxes[0]
        expect(selectAllCheckbox).toBeTruthy()

        const disabledCheckboxes = fixture.debugElement.queryAll(By.css('table .p-checkbox .p-checkbox-box.p-disabled'))
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
            expect(ch.nativeElement).withContext('checkbox should be selected').toHaveClass('p-highlight')
        }

        selectAllCheckbox.nativeElement.click() // deselect All unauthorized
        await fixture.whenStable()
        fixture.detectChanges()
        await fixture.whenStable() // Wait for PrimeNG to react on Select all change
        fixture.detectChanges()

        expect(component.selectedMachines.length).toEqual(0)
        for (const ch of checkboxes) {
            expect(ch.nativeElement).withContext('checkbox should not be selected').not.toHaveClass('p-highlight')
        }

        expect(selectAllCheckbox.nativeElement)
            .withContext('checkbox should not be selected')
            .not.toHaveClass('p-highlight')

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
        expect(selectAllCheckbox.nativeElement).withContext('checkbox should be selected').toHaveClass('p-highlight')

        // Deselect one machine.
        checkboxes[3].nativeElement.click()
        await fixture.whenStable()
        fixture.detectChanges()

        expect(component.selectedMachines.length).toEqual(2)
        expect(selectAllCheckbox.nativeElement)
            .withContext('checkbox should not be selected')
            .not.toHaveClass('p-highlight')
    })
})
