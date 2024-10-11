import { TestBed } from '@angular/core/testing'

import { Severity, VersionAlert, VersionService } from './version.service'
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
        getSwVersionsSpy = spyOn(generalService, 'getIscSwVersions')
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
        // Check if cache works, getIscSwVersions API should be only called once.
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
        // Check if isDataOutdated() works, getIscSwVersions API should be called again when
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
})
