import { TestBed } from '@angular/core/testing'

import { VersionService } from './version.service'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { AppsVersions, GeneralService } from './backend'
import { of } from 'rxjs'

describe('VersionService', () => {
    let service: VersionService
    let generalService: GeneralService

    beforeEach(() => {
        TestBed.configureTestingModule({
            providers: [],
            imports: [HttpClientTestingModule],
        })
        service = TestBed.inject(VersionService)
        generalService = TestBed.inject(GeneralService)
    })

    it('should be created', () => {
        expect(service).toBeTruthy()
    })

    it('should return current data', () => {
        // Arrange
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

        spyOn(generalService, 'getIscSwVersions').and.returnValue(of(fakeResponse as any))

        // Act
        let res1: AppsVersions
        let res2: AppsVersions
        // There is more than one observer.
        service.getCurrentData().subscribe((d) => (res1 = d))
        service.getCurrentData().subscribe((d) => (res2 = d))

        // Assert
        // Check if cache works, getIscSwVersions API should be only called once.
        expect(generalService.getIscSwVersions).toHaveBeenCalledTimes(1)
        expect(res1).toBeTruthy()
        expect(res2).toBeTruthy()
        expect(res1).toEqual(fakeResponse)
        expect(res2).toEqual(fakeResponse)
    })
})
