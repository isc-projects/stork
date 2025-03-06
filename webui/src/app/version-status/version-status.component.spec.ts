import { ComponentFixture, TestBed } from '@angular/core/testing'

import { VersionStatusComponent } from './version-status.component'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { TooltipModule } from 'primeng/tooltip'
import { Severity, VersionService } from '../version.service'
import { of } from 'rxjs'
import { MessagesModule } from 'primeng/messages'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'

describe('VersionStatusComponent', () => {
    let component: VersionStatusComponent
    let fixture: ComponentFixture<VersionStatusComponent>
    let versionService: VersionService
    let messageService: MessageService
    let getCurrentDataSpy: jasmine.Spy<any>
    let getSoftwareVersionFeedbackSpy: jasmine.Spy<any>
    let messageAddSpy: jasmine.Spy<any>
    const fakeResponse = {
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
            sortedStableVersions: ['9.18.30', '9.20.2'],
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
            sortedStableVersions: ['2.4.1', '2.6.1'],
        },
        stork: {
            currentStable: null,
            latestDev: { major: 1, minor: 19, releaseDate: '2024-10-02', status: 'Development', version: '1.19.0' },
            latestSecure: [
                {
                    major: 1,
                    minor: 15,
                    releaseDate: '2024-03-27',
                    status: 'Security update',
                    version: '1.15.1',
                },
            ],
            sortedStableVersions: null,
        },
    }

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [HttpClientTestingModule, TooltipModule, MessagesModule, BrowserAnimationsModule],
            declarations: [VersionStatusComponent],
            providers: [MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(VersionStatusComponent)
        versionService = TestBed.inject(VersionService)
        messageService = TestBed.inject(MessageService)
        component = fixture.componentInstance
        getCurrentDataSpy = spyOn(versionService, 'getCurrentData').and.returnValue(of(fakeResponse))
        getSoftwareVersionFeedbackSpy = spyOn(versionService, 'getSoftwareVersionFeedback').and.callThrough()
        messageAddSpy = spyOn(messageService, 'add').and.callThrough()
    })

    /**
     * Sets correct values of required input properties.
     */
    function setCorrectInputs() {
        fixture.componentRef.setInput('app', 'kea')
        fixture.componentRef.setInput('version', '2.6.1')
        fixture.detectChanges()
    }

    it('should create', () => {
        setCorrectInputs()
        expect(component).toBeTruthy()
    })

    it('should return app name', () => {
        // Arrange
        setCorrectInputs()

        // Act & Assert
        expect(component.appName).toBe('Kea')

        fixture.componentRef.setInput('app', 'stork')
        component.ngOnInit()
        fixture.detectChanges()
        expect(component.appName).toBe('Stork agent')

        fixture.componentRef.setInput('app', 'bind9')
        component.ngOnInit()
        fixture.detectChanges()
        expect(component.appName).toBe('BIND9')
    })

    it('should get current versions data', () => {
        // Arrange
        setCorrectInputs()

        // Act & Assert
        expect(getCurrentDataSpy).toHaveBeenCalledTimes(1)
        expect(getSoftwareVersionFeedbackSpy).toHaveBeenCalledOnceWith('2.6.1', 'kea', fakeResponse)
        expect(component.severity).toBe(Severity.success)
        expect(component.feedbackMessages).toBeTruthy()
        expect(component.feedbackMessages.length).toBeGreaterThan(0)
        expect(Object.keys(component.iconClasses).length).toBeGreaterThan(1)
    })

    it('should not get current versions data for invalid semver', () => {
        // Arrange
        fixture.componentRef.setInput('app', 'kea')
        fixture.componentRef.setInput('version', 'a.b.c')
        fixture.detectChanges()

        // Act & Assert
        expect(getCurrentDataSpy).toHaveBeenCalledTimes(0)
        expect(getSoftwareVersionFeedbackSpy).toHaveBeenCalledTimes(0)
        expect(messageAddSpy).toHaveBeenCalledOnceWith({
            severity: 'error',
            summary: 'Error parsing software version',
            detail: `Couldn't parse valid semver from given a.b.c version!`,
            life: 10000,
        })
        expect(component.severity).toBeUndefined()
        expect(component.feedbackMessages).toBeTruthy()
        expect(component.feedbackMessages.length).toBe(0)
        expect(Object.keys(component.iconClasses).length).toBe(0)
    })

    it('should display an error when version service observable emits error', () => {
        // Arrange
        getSoftwareVersionFeedbackSpy.and.throwError(new Error('internal error'))
        setCorrectInputs()

        // Act & Assert
        expect(getCurrentDataSpy).toHaveBeenCalledTimes(1)
        expect(getSoftwareVersionFeedbackSpy).toHaveBeenCalledTimes(1)
        expect(messageAddSpy).toHaveBeenCalledOnceWith({
            severity: 'error',
            summary: 'Error fetching software version data',
            detail: 'Error when fetching software version data: internal error',
            life: 10000,
        })
        expect(component.severity).toBeUndefined()
        expect(component.feedbackMessages).toBeTruthy()
        expect(component.feedbackMessages.length).toBe(0)
        expect(Object.keys(component.iconClasses).length).toBe(0)
    })

    it('should display app name', () => {
        // Arrange
        fixture.componentRef.setInput('showAppName', true)

        // Act & Assert
        setCorrectInputs()
        const span = fixture.nativeElement.querySelector('span')
        expect(span).toBeTruthy()
        expect(span.textContent).toContain('Kea 2.6.1')
    })

    it('should not display app name', () => {
        // Arrange
        // Act & Assert
        setCorrectInputs()
        const span = fixture.nativeElement.querySelector('span')
        expect(span).toBeNull()
    })

    it('should not display block message', () => {
        // Arrange
        // Act & Assert
        setCorrectInputs()
        const messagesDiv = fixture.nativeElement.querySelector('div.p-messages')
        expect(messagesDiv).toBeNull()
    })

    it('should display block message', () => {
        // Arrange
        fixture.componentRef.setInput('inline', false)

        // Act & Assert
        setCorrectInputs()
        const messagesDiv = fixture.nativeElement.querySelector('div.p-messages')
        expect(messagesDiv).toBeTruthy()
    })

    it('should not display feedback when version undefined', () => {
        // Arrange
        fixture.componentRef.setInput('version', undefined)
        fixture.componentRef.setInput('app', 'kea')
        fixture.detectChanges()

        // Act & Assert
        // No feedback messages are expected.
        expect(component.feedbackMessages.length).toEqual(0)
        // No error message should be emitted.
        expect(messageAddSpy).toHaveBeenCalledTimes(0)
    })
})
