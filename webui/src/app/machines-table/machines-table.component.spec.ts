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
import { of } from 'rxjs'
import { AppsVersions, ServicesService } from '../backend'
import { Severity, VersionService } from '../version.service'
import { VersionStatusComponent } from '../version-status/version-status.component'
import { LocaltimePipe } from '../pipes/localtime.pipe'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { TooltipModule } from 'primeng/tooltip'

describe('MachinesTableComponent', () => {
    let component: MachinesTableComponent
    let fixture: ComponentFixture<MachinesTableComponent>
    let versionServiceStub: Partial<VersionService>
    let servicesApi: any
    let getMachinesSpy: any

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
        fixture.detectChanges()

        // Do not save table state between tests, because that makes tests unstable.
        spyOn(component.table, 'saveState').and.callFake(() => {})
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should load all machines', async () => {
        // Arrange
        servicesApi.getUnauthorizedMachinesCount.and.returnValue(of(5))

        // Act
        component.loadData({ first: 0, rows: 10, filters: {} })
        await fixture.whenStable()
        fixture.detectChanges()

        // Assert
        expect(getMachinesSpy).toHaveBeenCalledOnceWith(0, 10, null, null, null)
        expect(component.dataCollection).toBe(getAllMachinesResp.items)
        expect(component.totalRecords).toBe(5)
        expect(servicesApi.getUnauthorizedMachinesCount).toHaveBeenCalledTimes(1)
        expect(component.unauthorizedMachinesCount).toBe(5)
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

    it('should apply filters when requesting data', async () => {})
})
