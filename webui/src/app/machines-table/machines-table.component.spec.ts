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
import { AppsVersions, ServicesService } from '../backend'
import { Severity, VersionService } from '../version.service'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { TooltipModule } from 'primeng/tooltip'
import { FilterMetadata } from 'primeng/api/filtermetadata'
import { deepCopy } from '../utils'
import objectContaining = jasmine.objectContaining

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

    beforeEach(async () => {
        // VersionService stub
        versionServiceStub = {
            sanitizeSemver: () => '1.2.3',
            getCurrentData: () => of({} as AppsVersions),
            getSoftwareVersionFeedback: () => ({ severity: Severity.success, messages: ['test feedback'] }),
        }

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
})
