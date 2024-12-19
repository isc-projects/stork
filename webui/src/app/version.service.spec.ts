import { TestBed } from '@angular/core/testing'

import { Severity, UpdateNotification, VersionAlert, VersionService } from './version.service'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { App, AppsVersions, GeneralService, VersionDetails } from './backend'
import { of } from 'rxjs'
import { deepCopy } from './utils'

describe('VersionService', () => {
    let service: VersionService
    let generalService: GeneralService
    let getSwVersionsSpy: jasmine.Spy<any>
    const fakeResponse: AppsVersions = {
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
        dataSource: 'offline',
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

    beforeEach(() => {
        TestBed.configureTestingModule({
            providers: [],
            imports: [HttpClientTestingModule],
        })
        service = TestBed.inject(VersionService)
        generalService = TestBed.inject(GeneralService)
        getSwVersionsSpy = spyOn(generalService, 'getSoftwareVersions')
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
        // Check if cache works, getSoftwareVersions API should be only called once.
        expect(getSwVersionsSpy).toHaveBeenCalledTimes(1)
        expect(res1).toBeTruthy()
        expect(res2).toBeTruthy()
        expect(res3).toBeTruthy()
        expect(JSON.stringify(res1)).toEqual(JSON.stringify(fakeResponse))
        expect(JSON.stringify(res2)).toEqual(JSON.stringify(fakeResponse))
        expect(JSON.stringify(res3)).toEqual(JSON.stringify(fakeResponse))
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
        expect(JSON.stringify(response)).toEqual(JSON.stringify(fakeResponse))
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
        const spy = spyOn(service, 'isDataOutdated')
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
        // Check if isDataOutdated() works, getSoftwareVersions API should be called again when
        // second observer subscribes. For third observer cache works because cached data is still valid.
        expect(getSwVersionsSpy).toHaveBeenCalledTimes(2)
        expect(res1).toBeTruthy()
        expect(res2).toBeTruthy()
        expect(res3).toBeTruthy()
        expect(JSON.stringify(res1)).toEqual(JSON.stringify(fakeResponse))
        expect(JSON.stringify(res2)).toEqual(JSON.stringify(fakeResponse))
        expect(JSON.stringify(res3)).toEqual(JSON.stringify(fakeResponse))
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
        let response: string

        // Act
        service
            .getDataSource()
            .subscribe((d) => (response = d))
            .unsubscribe()

        // Assert
        expect(getSwVersionsSpy).toHaveBeenCalledTimes(1)
        expect(response).toBeTruthy()
        expect(response).toEqual(AppsVersions.DataSourceEnum.Offline)
    })

    it('should sanitize semver', () => {
        // Arrange
        // Act
        const res1 = service.sanitizeSemver('there is semver here 12.23.1 where? there <=')
        const res2 = service.sanitizeSemver('BIND 9.18.30 (Extended Support Version) <id:cdc8d69>')
        const res3 = service.sanitizeSemver('2.6.3')

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
        // Act
        const securityUpdateFound = service.getSoftwareVersionFeedback('1.14.0', 'stork', fakeResponse)
        const stableUpdateFound = service.getSoftwareVersionFeedback('9.18.10', 'bind9', fakeResponse)
        const devUpdateFound = service.getSoftwareVersionFeedback('2.7.1', 'kea', fakeResponse)

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
        expect(securityUpdateFound.update).toBe('1.15.1')
        expect(stableUpdateFound.update).toBe('9.18.30')
        expect(devUpdateFound.update).toBe('2.7.3')
    })

    it('should return software version feedback for current version used', () => {
        // Arrange
        // Act
        const devNoStableCheck = service.getSoftwareVersionFeedback('1.19.0', 'stork', fakeResponse)
        const stableCheck = service.getSoftwareVersionFeedback('9.20.2', 'bind9', fakeResponse)
        const devCheck = service.getSoftwareVersionFeedback('2.7.3', 'kea', fakeResponse)

        // Assert
        expect(devNoStableCheck).toBeTruthy()
        expect(stableCheck).toBeTruthy()
        expect(devCheck).toBeTruthy()
        expect(devNoStableCheck.severity).toBe(Severity.success)
        expect(stableCheck.severity).toBe(Severity.success)
        expect(devCheck.severity).toBe(Severity.success)
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
        // Act
        const devNoStableCheck = service.getSoftwareVersionFeedback('1.19.5', 'stork', fakeResponse)
        const stableCheck = service.getSoftwareVersionFeedback('9.20.22', 'bind9', fakeResponse)
        const devCheck = service.getSoftwareVersionFeedback('2.9.3', 'kea', fakeResponse)

        // Assert
        expect(devNoStableCheck).toBeTruthy()
        expect(stableCheck).toBeTruthy()
        expect(devCheck).toBeTruthy()
        expect(devNoStableCheck.severity).toBe(Severity.secondary)
        expect(stableCheck.severity).toBe(Severity.secondary)
        expect(devCheck.severity).toBe(Severity.secondary)
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
        // Act
        const stableCheck = service.getSoftwareVersionFeedback('2.0.0', 'stork', fakeResponse)

        // Assert
        expect(stableCheck).toBeTruthy()
        expect(stableCheck.severity).toBe(Severity.secondary)
        expect(stableCheck.messages.length).toBe(1)
        expect(stableCheck.messages[0]).toMatch(new RegExp(/the .+ \d+.\d+.\d+ stable version is not known yet/))
    })

    it('should return software version feedback for newer stable not matching known ranges', () => {
        // Arrange
        // Act
        const stableCheck = service.getSoftwareVersionFeedback('4.0.0', 'kea', fakeResponse)

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
        // Act
        const stableCheck = service.getSoftwareVersionFeedback('1.0.0', 'kea', fakeResponse)

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
        const data: AppsVersions = {
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

        // Stable release metadata is there, but sortedStableVersions is missing.
        // It is expected to throw an error here as well.
        data.kea = { currentStable: [{ version: '2.6.1', releaseDate: '2024-02-01', eolDate: '2026-02-01' }] }
        expect(() => {
            service.getSoftwareVersionFeedback('2.7.1', 'kea', data)
        }).toThrowError("Couldn't asses the software version for Kea 2.7.1!")

        // Stable release metadata is missing.
        // It is expected to throw an error here as well.
        data.kea = { currentStable: [], sortedStableVersions: ['2.6.1'] }
        expect(() => {
            service.getSoftwareVersionFeedback('2.7.1', 'kea', data)
        }).toThrowError("Couldn't asses the software version for Kea 2.7.1!")
    })

    it('should throw error for version check for incomplete AppVersions data', () => {
        // Arrange
        const data: AppsVersions = {
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
                sortedStableVersions: null, // sortedStableVersions missing
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
        service.setStorkServerVersion('1.19.0')

        // Act
        const storkCheck = service.getSoftwareVersionFeedback('1.18.0', 'stork', fakeResponse)

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
        const f = service.getSoftwareVersionFeedback('2.4.1', 'kea', fakeResponse)
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

    it('should detect mismatching kea daemons', () => {
        // Arrange & Act & Assert
        // Test some cases with incomplete data.
        expect(service.areKeaDaemonsVersionsMismatching(null)).toBeFalsy()
        expect(service.areKeaDaemonsVersionsMismatching(undefined)).toBeFalsy()
        expect(service.areKeaDaemonsVersionsMismatching({})).toBeFalsy()
        expect(service.areKeaDaemonsVersionsMismatching({ type: 'bind9' } as App)).toBeFalsy()
        expect(service.areKeaDaemonsVersionsMismatching({ type: 'kea' } as App)).toBeFalsy()
        expect(service.areKeaDaemonsVersionsMismatching({ type: 'kea', details: {} } as App)).toBeFalsy()
        expect(service.areKeaDaemonsVersionsMismatching({ type: 'kea', details: { daemons: null } } as App)).toBeFalsy()
        expect(service.areKeaDaemonsVersionsMismatching({ type: 'kea', details: { daemons: [] } } as App)).toBeFalsy()
        expect(
            service.areKeaDaemonsVersionsMismatching({
                type: 'kea',
                details: { daemons: [{ id: 1 }, { id: 2 }] },
            } as App)
        ).toBeFalsy()
        expect(
            service.areKeaDaemonsVersionsMismatching({
                type: 'kea',
                details: { daemons: [{ id: 1 }, { id: 2, version: '2.6.1' }] },
            } as App)
        ).toBeFalsy()
        // Test valid data.
        expect(
            service.areKeaDaemonsVersionsMismatching({
                type: 'kea',
                details: {
                    daemons: [
                        { id: 1, version: '2.6.1' },
                        { id: 2, version: '2.6.1' },
                    ],
                },
            } as App)
        ).toBeFalsy()
        expect(
            service.areKeaDaemonsVersionsMismatching({
                type: 'kea',
                details: {
                    daemons: [{ id: 1 }, { id: 2, version: '2.6.1' }, { id: 3, version: '2.6.1' }],
                },
            } as App)
        ).toBeFalsy()
        expect(
            service.areKeaDaemonsVersionsMismatching({
                type: 'kea',
                details: {
                    daemons: [
                        { id: 1, version: '2.6.1' },
                        { id: 2, version: '2.6.1' },
                        { id: 3, version: '2.6.0' },
                    ],
                },
            } as App)
        ).toBeTrue()
        expect(
            service.areKeaDaemonsVersionsMismatching({
                type: 'kea',
                details: {
                    daemons: [{ id: 1 }, { id: 2, version: '2.6.1' }, { id: 3, version: '2.6.0' }],
                },
            } as App)
        ).toBeTrue()
    })

    it('should detect alerting severity', () => {
        // Arrange & Act & Assert
        let alert: VersionAlert
        service.getVersionAlert().subscribe((a) => (alert = a))

        // Check initial alert.
        expect(alert.detected).toBeFalse()
        expect(alert.severity).toEqual(Severity.success)

        // Severity less severe than Warning should not have any effect on the alert.
        service.detectAlertingSeverity(Severity.success)
        expect(alert.detected).toBeFalse()
        expect(alert.severity).toEqual(Severity.success)

        service.detectAlertingSeverity(Severity.secondary)
        expect(alert.detected).toBeFalse()
        expect(alert.severity).toEqual(Severity.success)

        service.detectAlertingSeverity(Severity.info)
        expect(alert.detected).toBeFalse()
        expect(alert.severity).toEqual(Severity.success)

        // Warning severity should trigger the alert.
        service.detectAlertingSeverity(Severity.warn)
        expect(alert.detected).toBeTrue()
        expect(alert.severity).toEqual(Severity.warn)

        // Error severity should raise the alert level.
        service.detectAlertingSeverity(Severity.error)
        expect(alert.detected).toBeTrue()
        expect(alert.severity).toEqual(Severity.error)

        // Lower severity should not have any effect on the alert.
        service.detectAlertingSeverity(Severity.warn)
        expect(alert.detected).toBeTrue()
        expect(alert.severity).toEqual(Severity.error)

        service.detectAlertingSeverity(Severity.success)
        expect(alert.detected).toBeTrue()
        expect(alert.severity).toEqual(Severity.error)
    })

    it('should return more recent dev than a stable', () => {
        // Arrange & Act & Assert
        // There is no stable stork release in fakeResponse data, so true is expected.
        expect(service.isDevMoreRecentThanStable('stork', fakeResponse)).toBeTrue()
        // Kea has more recent dev release than a stable release.
        expect(service.isDevMoreRecentThanStable('kea', fakeResponse)).toBeTrue()
        // BIND9 has more recent dev release than a stable release.
        expect(service.isDevMoreRecentThanStable('bind9', fakeResponse)).toBeTrue()
        const data = deepCopy(fakeResponse)
        data.stork.currentStable = [
            {
                eolDate: '2026-07-01',
                major: 2,
                minor: 0,
                range: '2.0.x',
                releaseDate: '2024-11-13',
                status: 'Current Stable',
                version: '2.0.0',
            },
        ]
        data.stork.sortedStableVersions = ['2.0.0']
        data.kea.latestDev = {} as VersionDetails
        // In this data, Stork stable is more recent than the dev release, so false is expected.
        expect(service.isDevMoreRecentThanStable('stork', data)).toBeFalse()
        // In this data, Kea has no dev release, so false is expected.
        expect(service.isDevMoreRecentThanStable('kea', data)).toBeFalse()
    })

    it('should return software version feedback for security updates available', () => {
        // Arrange
        const data = deepCopy(fakeResponse)
        data.kea.latestSecure = [
            { version: '2.6.10', range: '2.6.x', releaseDate: '2024-12-02' },
            { version: '2.4.10', range: '2.4.x', releaseDate: '2024-12-02' },
            { version: '2.7.10', range: '2.7.x', releaseDate: '2024-12-02' },
        ]

        // Act
        const securityCheckCurrentStable1 = service.getSoftwareVersionFeedback('2.4.1', 'kea', data)
        const securityCheckCurrentStable2 = service.getSoftwareVersionFeedback('2.6.1', 'kea', data)
        const securityCheckCurrentDev = service.getSoftwareVersionFeedback('2.7.5', 'kea', data)
        const securityCheckOlderDev = service.getSoftwareVersionFeedback('2.5.15', 'kea', data)

        // Assert
        // Matches latestSecure range and is older.
        expect(securityCheckCurrentStable1).toBeTruthy()
        expect(securityCheckCurrentStable1.severity).toBe(Severity.error)
        expect(securityCheckCurrentStable1.messages.length).toBe(1)
        expect(securityCheckCurrentStable1.messages[0]).toMatch(new RegExp(/Security update \d+.\d+.\d+ was released/))

        // Matches latestSecure range and is older.
        expect(securityCheckCurrentStable2).toBeTruthy()
        expect(securityCheckCurrentStable2.severity).toBe(Severity.error)
        expect(securityCheckCurrentStable2.messages.length).toBe(1)
        expect(securityCheckCurrentStable2.messages[0]).toMatch(new RegExp(/Security update \d+.\d+.\d+ was released/))

        // Matches latestSecure range + it's dev release and is older.
        expect(securityCheckCurrentDev).toBeTruthy()
        expect(securityCheckCurrentDev.severity).toBe(Severity.error)
        expect(securityCheckCurrentDev.messages.length).toBe(1)
        expect(securityCheckCurrentDev.messages[0]).toMatch(new RegExp(/Security update \d+.\d+.\d+ was released/))

        // Does not match latestSecure range, but it's dev release and is older.
        // It is considered insecure.
        expect(securityCheckOlderDev).toBeTruthy()
        expect(securityCheckOlderDev.severity).toBe(Severity.error)
        expect(securityCheckOlderDev.messages.length).toBe(1)
        expect(securityCheckOlderDev.messages[0]).toMatch(new RegExp(/Security update \d+.\d+.\d+ was released/))
    })

    it('should not return software version feedback for security updates available', () => {
        // Arrange
        const data = deepCopy(fakeResponse)
        data.kea.latestSecure = [
            { version: '2.6.10', range: '2.6.x', releaseDate: '2024-12-02' },
            { version: '2.4.10', range: '2.4.x', releaseDate: '2024-12-02' },
            { version: '2.7.10', range: '2.7.x', releaseDate: '2024-12-02' },
        ]
        data.kea.currentStable[0].version = '2.6.10'
        data.kea.currentStable[1].version = '2.4.10'
        data.kea.latestDev.version = '2.7.10'

        // Act
        const securityCheckCurrentStable1 = service.getSoftwareVersionFeedback('2.4.10', 'kea', data)
        const securityCheckCurrentStable2 = service.getSoftwareVersionFeedback('2.6.10', 'kea', data)
        const securityCheckOlderStable = service.getSoftwareVersionFeedback('2.2.1', 'kea', data)
        const securityCheckCurrentDev = service.getSoftwareVersionFeedback('2.7.10', 'kea', data)
        const securityCheckNewerDev1 = service.getSoftwareVersionFeedback('3.1.0', 'kea', data)
        const securityCheckNewerDev2 = service.getSoftwareVersionFeedback('2.7.11', 'kea', data)

        // Assert
        // Matches latestSecure range and it is equal to latest stable release.
        expect(securityCheckCurrentStable1).toBeTruthy()
        expect(securityCheckCurrentStable1.severity).toBe(Severity.success)
        expect(securityCheckCurrentStable1.messages.length).toBe(1)
        expect(securityCheckCurrentStable1.messages[0]).toMatch(new RegExp(/\d+.\d+.\d+ is current .+ stable version/))

        // Matches other latestSecure range and it is equal to latest stable release.
        expect(securityCheckCurrentStable2).toBeTruthy()
        expect(securityCheckCurrentStable2.severity).toBe(Severity.success)
        expect(securityCheckCurrentStable2.messages.length).toBe(1)
        expect(securityCheckCurrentStable2.messages[0]).toMatch(new RegExp(/\d+.\d+.\d+ is current .+ stable version/))

        // Does not match latestSecure range. Detected stable minor is below latestSecure ranges.
        // It is considered secure.
        expect(securityCheckOlderStable).toBeTruthy()
        expect(securityCheckOlderStable.severity).toBe(Severity.warn)
        expect(securityCheckOlderStable.messages.length).toBe(1)
        expect(securityCheckOlderStable.messages[0]).toMatch(
            new RegExp(/version \d+.\d+.\d+ is older than current stable version/)
        )

        // Matches latestSecure range and it is equal to latest development release.
        expect(securityCheckCurrentDev).toBeTruthy()
        expect(securityCheckCurrentDev.severity).toBe(Severity.success)
        expect(securityCheckCurrentDev.messages.length).toBe(2)
        expect(securityCheckCurrentDev.messages[0]).toMatch(new RegExp(/\d+.\d+.\d+ is current .+ development version/))
        expect(securityCheckCurrentDev.messages[1]).toMatch(
            'Please be advised that using development version in production is not recommended'
        )

        // Does not match latestSecure range. Dev release detected and it is more recent than the latest development release.
        expect(securityCheckNewerDev1).toBeTruthy()
        expect(securityCheckNewerDev1.severity).toBe(Severity.secondary)
        expect(securityCheckNewerDev1.messages.length).toBe(2)
        expect(securityCheckNewerDev1.messages[0]).toMatch('You are using more recent version')
        expect(securityCheckNewerDev1.messages[1]).toMatch(
            'Please be advised that using development version in production is not recommended'
        )

        // Matches latestSecure range and it is more recent than the latest development release.
        expect(securityCheckNewerDev2).toBeTruthy()
        expect(securityCheckNewerDev2.severity).toBe(Severity.secondary)
        expect(securityCheckNewerDev2.messages.length).toBe(2)
        expect(securityCheckNewerDev2.messages[0]).toMatch('You are using more recent version')
        expect(securityCheckNewerDev2.messages[1]).toMatch(
            'Please be advised that using development version in production is not recommended'
        )
    })

    it('should return software version feedback when dev metadata is missing', () => {
        // Arrange
        const data = deepCopy(fakeResponse)
        data.kea.latestDev = null

        // Act
        const devUpdateFound = service.getSoftwareVersionFeedback('2.7.1', 'kea', data)

        // Assert
        expect(devUpdateFound).toBeTruthy()
        expect(devUpdateFound.severity).toBe(Severity.secondary)
        expect(devUpdateFound.messages.length).toBeGreaterThan(1)
        expect(devUpdateFound.messages[0]).toMatch(new RegExp(/has current stable version.+available/))
        expect(devUpdateFound.messages[1]).toMatch(
            'Please be advised that using development version in production is not recommended'
        )
    })

    it('should emit notification about stork server update', () => {
        // Arrange
        let notification: UpdateNotification
        service.getStorkServerUpdateNotification().subscribe((n) => (notification = n))
        expect(notification).toBeTruthy()
        expect(notification.available).toBeFalse()
        // Stork server version not set yet, so there should be no notification.
        service.getCurrentData().subscribe().unsubscribe()
        expect(notification.available).toBeFalse()

        // Act
        service.setStorkServerVersion('1.18.0')
        service.refreshData()

        // Assert
        expect(notification).toBeTruthy()
        expect(notification.available).toBeTrue()
        expect(notification.feedback).toBeTruthy()
        expect(notification.feedback.update).toBe('1.19.0')
        expect(notification.feedback.severity).toBe(Severity.warn)
        expect(notification.feedback.messages).toBeTruthy()
        expect(notification.feedback.messages.length).toBeGreaterThan(0)
        expect(notification.feedback.messages[0]).toContain('Stork server update is available')
    })

    it('should emit notification about stork server security update', () => {
        // Arrange
        let notification: UpdateNotification
        service.getStorkServerUpdateNotification().subscribe((n) => (notification = n))
        service.setStorkServerVersion('1.15.0')

        // Act
        service.getCurrentData().subscribe().unsubscribe()

        // Assert
        expect(notification).toBeTruthy()
        expect(notification.available).toBeTrue()
        expect(notification.feedback).toBeTruthy()
        expect(notification.feedback.update).toBe('1.15.1')
        expect(notification.feedback.severity).toBe(Severity.error)
        expect(notification.feedback.messages).toBeTruthy()
        expect(notification.feedback.messages.length).toBeGreaterThan(0)
        expect(notification.feedback.messages[0]).toContain('Stork server security update is available')
    })

    it('should not emit notification about stork server update', () => {
        // Arrange
        service.setStorkServerVersion('1.19.0')
        let notification: UpdateNotification
        service.getStorkServerUpdateNotification().subscribe((n) => (notification = n))
        expect(notification).toBeTruthy()
        expect(notification.available).toBeFalse()

        // Act
        service.getCurrentData().subscribe().unsubscribe()

        // Assert
        expect(notification).toBeTruthy()
        expect(notification.available).toBeFalse()
    })
})
