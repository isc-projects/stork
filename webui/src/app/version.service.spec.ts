import { TestBed } from '@angular/core/testing'

import { Severity, VersionAlert, VersionFeedback, VersionService } from './version.service'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { AppsVersions, GeneralService } from './backend'
import { of } from 'rxjs'

describe('VersionService', () => {
    let service: VersionService
    let generalService: GeneralService
    let getSwVersionsSpy: jasmine.Spy<any>
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

    beforeEach(() => {
        TestBed.configureTestingModule({
            providers: [],
            imports: [HttpClientTestingModule],
        })
        service = TestBed.inject(VersionService)
        generalService = TestBed.inject(GeneralService)
        getSwVersionsSpy = spyOn(generalService, 'getISCSoftwareVersions')
        getSwVersionsSpy.and.returnValue(of(fakeResponse))
    })

    it('should be created', () => {
        expect(service).toBeTruthy()
    })

    it('should return current data', () => {
        // Arrange
        let res1: AppsVersions
        let res2: AppsVersions
        let res3: AppsVersions

        // Act
        // There is more than one observer subscribed.
        service
            .getCurrentData()
            .subscribe((d) => (res1 = d))
            .unsubscribe()
        service
            .getCurrentData()
            .subscribe((d) => (res2 = d))
            .unsubscribe()
        service
            .getCurrentData()
            .subscribe((d) => (res3 = d))
            .unsubscribe()

        // Assert
        // Check if cache works, getISCSoftwareVersions API should be only called once.
        expect(getSwVersionsSpy).toHaveBeenCalledTimes(1)
        expect(res1).toBeTruthy()
        expect(res2).toBeTruthy()
        expect(res3).toBeTruthy()
        expect(res1).toEqual(fakeResponse)
        expect(res2).toEqual(fakeResponse)
        expect(res3).toEqual(fakeResponse)
    })

    it('should query api when data refresh is forced', () => {
        // Arrange
        let response: AppsVersions
        service
            .getCurrentData()
            .subscribe((d) => (response = d))
            .unsubscribe()

        // Act
        service.refreshData()

        // Assert
        expect(getSwVersionsSpy).toHaveBeenCalledTimes(2)
        expect(response).toBeTruthy()
        expect(response).toEqual(fakeResponse)
    })

    it('should refresh outdated data', () => {
        // Arrange
        let res1: AppsVersions
        let res2: AppsVersions
        let res3: AppsVersions
        service
            .getCurrentData()
            .subscribe((d) => (res1 = d))
            .unsubscribe()

        // Act
        let spy = spyOn(service, 'isDataOutdated')
        spy.and.returnValue(true)
        service
            .getCurrentData()
            .subscribe((d) => (res2 = d))
            .unsubscribe()
        spy.and.callThrough()
        service
            .getCurrentData()
            .subscribe((d) => (res3 = d))
            .unsubscribe()

        // Assert
        // Check if isDataOutdated() works, getISCSoftwareVersions API should be called again when
        // second observer subscribes. For third observer cache works because cached data is still valid.
        expect(getSwVersionsSpy).toHaveBeenCalledTimes(2)
        expect(res1).toBeTruthy()
        expect(res2).toBeTruthy()
        expect(res3).toBeTruthy()
        expect(res1).toEqual(fakeResponse)
        expect(res2).toEqual(fakeResponse)
        expect(res3).toEqual(fakeResponse)
    })

    it('should return data manufacture date', () => {
        // Arrange
        let response: string

        // Act
        service
            .getDataManufactureDate()
            .subscribe((d) => (response = d))
            .unsubscribe()

        // Assert
        expect(getSwVersionsSpy).toHaveBeenCalledTimes(1)
        expect(response).toBeTruthy()
        expect(response).toEqual('2024-10-03')
    })

    it('should return online data flag', () => {
        // Arrange
        let response: boolean

        // Act
        service
            .isOnlineData()
            .subscribe((d) => (response = d))
            .unsubscribe()

        // Assert
        expect(getSwVersionsSpy).toHaveBeenCalledTimes(1)
        expect(response).not.toBeNull()
        expect(response).toBeFalse()
    })

    it('should sanitize semver', () => {
        // Arrange
        // Act
        let res1 = service.sanitizeSemver('there is semver here 12.23.1 where? there <=')
        let res2 = service.sanitizeSemver('BIND 9.18.30 (Extended Support Version) <id:cdc8d69>')
        let res3 = service.sanitizeSemver('2.6.3')

        // Assert
        expect(service.sanitizeSemver(null)).toBeNull()
        expect(service.sanitizeSemver(undefined)).toBeNull()
        expect(service.sanitizeSemver('foobar')).toBeNull()
        expect(service.sanitizeSemver('a.b.c')).toBeNull()
        expect(res1).toEqual('12.23.1')
        expect(res2).toEqual('9.18.30')
        expect(res3).toEqual('2.6.3')
    })

    it('should not emit version alert when there was no warning nor error severity detected', () => {
        // Arrange
        let resp: VersionAlert
        // Act
        service
            .getVersionAlert()
            .subscribe((d) => (resp = d))
            .unsubscribe()
        // Assert
        expect(resp).toBeTruthy()
        expect(resp.detected).toBeFalse()
        expect(resp.severity).toBe(Severity.success)
    })

    it('should return software version feedback for update available', () => {
        // Arrange
        let securityUpdateFound: VersionFeedback
        let stableUpdateFound: VersionFeedback
        let devUpdateFound: VersionFeedback

        // Act
        securityUpdateFound = service.getSoftwareVersionFeedback('1.14.0', 'stork', fakeResponse)
        stableUpdateFound = service.getSoftwareVersionFeedback('9.18.10', 'bind9', fakeResponse)
        devUpdateFound = service.getSoftwareVersionFeedback('2.7.1', 'kea', fakeResponse)

        // Assert
        expect(securityUpdateFound).toBeTruthy()
        expect(stableUpdateFound).toBeTruthy()
        expect(devUpdateFound).toBeTruthy()
        expect(securityUpdateFound.severity).toBe(Severity.error)
        expect(stableUpdateFound.severity).toBe(Severity.info)
        expect(devUpdateFound.severity).toBe(Severity.warn)
        expect(securityUpdateFound.messages.length).toBeGreaterThan(0)
        expect(stableUpdateFound.messages.length).toBeGreaterThan(0)
        expect(devUpdateFound.messages.length).toBeGreaterThan(1)
        expect(securityUpdateFound.messages[0]).toMatch(new RegExp(/Security update \d+.\d+.\d+ was released/))
        expect(stableUpdateFound.messages[0]).toMatch(
            new RegExp(/Stable .+ version update \(\d+.\d+.\d+\) is available/)
        )
        expect(devUpdateFound.messages[0]).toMatch(
            new RegExp(/Development .+ version update \(\d+.\d+.\d+\) is available/)
        )
        expect(devUpdateFound.messages[1]).toMatch(
            'Please be advised that using development version in production is not recommended'
        )
    })

    it('should return software version feedback for current version used', () => {
        // Arrange
        let devNoStableCheck: VersionFeedback
        let stableCheck: VersionFeedback
        let devCheck: VersionFeedback

        // Act
        devNoStableCheck = service.getSoftwareVersionFeedback('1.19.0', 'stork', fakeResponse)
        stableCheck = service.getSoftwareVersionFeedback('9.20.2', 'bind9', fakeResponse)
        devCheck = service.getSoftwareVersionFeedback('2.7.3', 'kea', fakeResponse)

        // Assert
        expect(devNoStableCheck).toBeTruthy()
        expect(stableCheck).toBeTruthy()
        expect(devCheck).toBeTruthy()
        expect(devNoStableCheck.severity).toBe(Severity.success)
        expect(stableCheck.severity).toBe(Severity.success)
        expect(devCheck.severity).toBe(Severity.warn)
        expect(devNoStableCheck.messages.length).toBe(1)
        expect(stableCheck.messages.length).toBe(1)
        expect(devCheck.messages.length).toBeGreaterThan(1)
        expect(devNoStableCheck.messages[0]).toMatch(new RegExp(/\d+.\d+.\d+ is current .+ development version/))
        expect(stableCheck.messages[0]).toMatch(new RegExp(/\d+.\d+.\d+ is current .+ stable version/))
        expect(devCheck.messages[0]).toMatch(new RegExp(/\d+.\d+.\d+ is current .+ development version/))
        expect(devCheck.messages[1]).toMatch(
            'Please be advised that using development version in production is not recommended'
        )
    })

    it('should return software version feedback for more recent version used', () => {
        // Arrange
        let devNoStableCheck: VersionFeedback
        let stableCheck: VersionFeedback
        let devCheck: VersionFeedback

        // Act
        devNoStableCheck = service.getSoftwareVersionFeedback('1.19.5', 'stork', fakeResponse)
        stableCheck = service.getSoftwareVersionFeedback('9.20.22', 'bind9', fakeResponse)
        devCheck = service.getSoftwareVersionFeedback('2.9.3', 'kea', fakeResponse)

        // Assert
        expect(devNoStableCheck).toBeTruthy()
        expect(stableCheck).toBeTruthy()
        expect(devCheck).toBeTruthy()
        expect(devNoStableCheck.severity).toBe(Severity.secondary)
        expect(stableCheck.severity).toBe(Severity.secondary)
        expect(devCheck.severity).toBe(Severity.warn)
        expect(devNoStableCheck.messages.length).toBe(1)
        expect(stableCheck.messages.length).toBe(1)
        expect(devCheck.messages.length).toBeGreaterThan(1)
        expect(devNoStableCheck.messages[0]).toMatch('You are using more recent version')
        expect(stableCheck.messages[0]).toMatch('You are using more recent version')
        expect(devCheck.messages[0]).toMatch('You are using more recent version')
        expect(devCheck.messages[1]).toMatch(
            'Please be advised that using development version in production is not recommended'
        )
    })

    it('should return software version feedback for not known stable', () => {
        // Arrange
        let stableCheck: VersionFeedback

        // Act
        stableCheck = service.getSoftwareVersionFeedback('2.0.0', 'stork', fakeResponse)

        // Assert
        expect(stableCheck).toBeTruthy()
        expect(stableCheck.severity).toBe(Severity.secondary)
        expect(stableCheck.messages.length).toBe(1)
        expect(stableCheck.messages[0]).toMatch(new RegExp(/the .+ \d+.\d+.\d+ stable version is not known yet/))
    })

    it('should return software version feedback for newer stable not matching known ranges', () => {
        // Arrange
        let stableCheck: VersionFeedback

        // Act
        stableCheck = service.getSoftwareVersionFeedback('4.0.0', 'kea', fakeResponse)

        // Assert
        expect(stableCheck).toBeTruthy()
        expect(stableCheck.severity).toBe(Severity.secondary)
        expect(stableCheck.messages.length).toBe(1)
        expect(stableCheck.messages[0]).toMatch(
            new RegExp(/version \d+.\d+.\d+ is more recent than current stable version/)
        )
    })

    it('should return software version feedback for older stable not matching known ranges', () => {
        // Arrange
        let stableCheck: VersionFeedback

        // Act
        stableCheck = service.getSoftwareVersionFeedback('1.0.0', 'kea', fakeResponse)

        // Assert
        expect(stableCheck).toBeTruthy()
        expect(stableCheck.severity).toBe(Severity.warn)
        expect(stableCheck.messages.length).toBe(1)
        expect(stableCheck.messages[0]).toMatch(new RegExp(/version \d+.\d+.\d+ is older than current stable version/))
    })

    it('should throw error for version check for incorrect semver', () => {
        // Arrange
        // Act
        // Assert
        expect(() => {
            service.getSoftwareVersionFeedback('a.b.c', 'kea', fakeResponse)
        }).toThrowError("Couldn't parse valid semver from given a.b.c version!")
    })

    it('should throw error for version check for incorrect AppVersions data', () => {
        // Arrange
        let data: AppsVersions = {
            date: 'abc',
            bind9: null,
            kea: null,
            stork: null,
        }
        // Act
        // Assert
        expect(() => {
            service.getSoftwareVersionFeedback('2.7.1', 'kea', data)
        }).toThrowError("Couldn't asses the software version for Kea 2.7.1!")
    })

    it('should throw error for version check for incomplete AppVersions data', () => {
        // Arrange
        let data: AppsVersions = {
            date: 'abc',
            bind9: null,
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
                sortedStables: null, // sortedStables missing
            },
            stork: null,
        }
        // Act
        // Assert
        expect(() => {
            service.getSoftwareVersionFeedback('4.0.0', 'kea', data)
        }).toThrowError('Invalid syntax of the software versions metadata JSON file received from Stork server.')
    })

    it('should return software version feedback for stork agent vs server mismatch', () => {
        // Arrange
        let storkCheck: VersionFeedback
        service.setStorkServerVersion('1.19.0')

        // Act
        storkCheck = service.getSoftwareVersionFeedback('1.18.0', 'stork', fakeResponse)

        // Assert
        expect(storkCheck).toBeTruthy()
        expect(storkCheck.severity).toBe(Severity.warn)
        expect(storkCheck.messages.length).toBe(2)
        expect(storkCheck.messages[0]).toMatch(new RegExp(/Development .+ version update \(\d+.\d+.\d+\) is available/))
        expect(storkCheck.messages[1]).toMatch(
            new RegExp(/Stork server \d+.\d+.\d+ and Stork agent \d+.\d+.\d+ versions do not match/)
        )
    })

    it('should emit version alert when there was warning or error severity detected', () => {
        // Arrange
        let resp: VersionAlert
        // Act
        service.getVersionAlert().subscribe((d) => (resp = d))
        // Assert
        expect(resp).toBeTruthy()
        expect(resp.detected).toBeFalse()
        expect(resp.severity).toBe(Severity.success)
        service.getSoftwareVersionFeedback('2.7.0', 'kea', fakeResponse)
        expect(resp.detected).toBeTrue()
        expect(resp.severity).toBe(Severity.warn)
        service.getSoftwareVersionFeedback('1.14.0', 'stork', fakeResponse)
        expect(resp.detected).toBeTrue()
        expect(resp.severity).toBe(Severity.error)
    })

    it('should dismiss version alert', () => {
        // Arrange
        let resp: VersionAlert
        let resp2: VersionAlert

        // Act
        service.getVersionAlert().subscribe((d) => (resp = d))
        expect(resp).toBeTruthy()
        expect(resp.detected).toBeFalse()
        expect(resp.severity).toBe(Severity.success)
        service.getSoftwareVersionFeedback('2.7.0', 'kea', fakeResponse)
        expect(resp.detected).toBeTrue()
        expect(resp.severity).toBe(Severity.warn)
        // success severity expected
        let f = service.getSoftwareVersionFeedback('2.4.1', 'kea', fakeResponse)
        expect(f.severity).toBe(Severity.success)
        // however this shouldn't change the alert severity
        expect(resp.detected).toBeTrue()
        expect(resp.severity).toBe(Severity.warn)

        service.dismissVersionAlert()

        // Assert
        expect(resp.detected).toBeFalse()
        expect(resp.severity).toBe(Severity.success)
        // Second observer should not receive anything after the alert has been dismissed.
        service.getVersionAlert().subscribe((d) => (resp2 = d))
        expect(resp2).toBeUndefined()
        // Detecting warning severity after the alert was dismissed should not enable the alert again.
        service.getSoftwareVersionFeedback('2.7.0', 'kea', fakeResponse)
        expect(resp.detected).toBeFalse()
        expect(resp.severity).toBe(Severity.success)
        expect(resp2).toBeUndefined()
    })
})
