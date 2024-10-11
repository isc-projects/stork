import { ComponentFixture, TestBed } from '@angular/core/testing'

import { VersionStatusComponent } from './version-status.component'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { TooltipModule } from 'primeng/tooltip'
import { Severity, VersionService } from '../version.service'
import { of } from 'rxjs'

describe('VersionStatusComponent', () => {
    let component: VersionStatusComponent
    let fixture: ComponentFixture<VersionStatusComponent>
    let versionService: VersionService
    let getCurrentDataSpy: jasmine.Spy<any>
    let getSoftwareVersionFeedback: jasmine.Spy<any>
    let fakeResponse = {
        bind9: {
            currentStable: [
                {
                    eolDate: '2026-07-01',
                    esv: 'true',
                    major: 9,
                    minor: 18,
                    range: '9.18.x',
                    releaseDate: '2024-09-18',
                    status: 'Current Stable',
                    version: '9.18.30',
                },
                {
                    eolDate: '2028-07-01',
                    major: 9,
                    minor: 20,
                    range: '9.20.x',
                    releaseDate: '2024-09-18',
                    status: 'Current Stable',
                    version: '9.20.2',
                },
            ],
            latestDev: { major: 9, minor: 21, releaseDate: '2024-09-18', status: 'Development', version: '9.21.1' },
            sortedStables: ['9.18.30', '9.20.2'],
        },
        date: '2024-10-03',
        kea: {
            currentStable: [
                {
                    eolDate: '2026-07-01',
                    major: 2,
                    minor: 6,
                    range: '2.6.x',
                    releaseDate: '2024-07-31',
                    status: 'Current Stable',
                    version: '2.6.1',
                },
                {
                    eolDate: '2025-07-01',
                    major: 2,
                    minor: 4,
                    range: '2.4.x',
                    releaseDate: '2023-11-29',
                    status: 'Current Stable',
                    version: '2.4.1',
                },
            ],
            latestDev: { major: 2, minor: 7, releaseDate: '2024-09-25', status: 'Development', version: '2.7.3' },
            sortedStables: ['2.4.1', '2.6.1'],
        },
        stork: {
            currentStable: null,
            latestDev: { major: 1, minor: 19, releaseDate: '2024-10-02', status: 'Development', version: '1.19.0' },
            latestSecure: {
                major: 1,
                minor: 15,
                releaseDate: '2024-03-27',
                status: 'Security update',
                version: '1.15.1',
            },
            sortedStables: null,
        },
    }

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [HttpClientTestingModule, TooltipModule],
            declarations: [VersionStatusComponent],
            providers: [MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(VersionStatusComponent)
        versionService = TestBed.inject(VersionService)
        component = fixture.componentInstance
        getCurrentDataSpy = spyOn(versionService, 'getCurrentData')
        getCurrentDataSpy.and.returnValue(of(fakeResponse))
        getSoftwareVersionFeedback = spyOn(versionService, 'getSoftwareVersionFeedback')
        getSoftwareVersionFeedback.and.callThrough()
        fixture.componentRef.setInput('app', 'kea')
        fixture.componentRef.setInput('version', '2.6.1')
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should return app name', () => {
        // Arrange
        // Act
        // Assert
        expect(component.appName).toBe('Kea')

        fixture.componentRef.setInput('app', 'stork')
        fixture.detectChanges()
        component.ngOnInit()
        fixture.detectChanges()
        expect(component.appName).toBe('Stork agent')

        fixture.componentRef.setInput('app', 'bind9')
        fixture.detectChanges()
        component.ngOnInit()
        fixture.detectChanges()
        expect(component.appName).toBe('Bind9')
    })

    it('should get current versions data', () => {
        // Arrange
        // Act
        // Assert
        expect(getCurrentDataSpy).toHaveBeenCalledTimes(1)
        expect(getSoftwareVersionFeedback).toHaveBeenCalledOnceWith('2.6.1', 'kea', fakeResponse)
        expect(component.severity).toBe(Severity.success)
        expect(component.feedbackMessages).toBeTruthy()
        expect(component.feedbackMessages.length).toBeGreaterThan(0)
    })
})
