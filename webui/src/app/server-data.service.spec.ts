import { fakeAsync, flush, TestBed } from '@angular/core/testing'
import { ServerDataService } from './server-data.service'
import { HttpClient, HttpErrorResponse, HttpHandler } from '@angular/common/http'
import { Router } from '@angular/router'
import { ServicesService, UsersService } from './backend'
import { MessageService } from 'primeng/api'
import { of, throwError } from 'rxjs'
import { AuthService } from './auth.service'

describe('ServerDataService', () => {
    let service: ServerDataService

    beforeEach(() => {
        TestBed.configureTestingModule({
            providers: [
                HttpClient,
                HttpHandler,
                UsersService,
                MessageService,
                ServicesService,
                {
                    provide: Router,
                    useValue: {},
                },
                {
                    provide: AuthService,
                    useValue: {
                        currentUser: of({}),
                    },
                },
            ],
        })
        service = TestBed.inject(ServerDataService)
    })

    it('should be created', () => {
        expect(service).toBeTruthy()
    })

    it('should return machine addresses', () => {
        const fakeResponse: any = {
            items: [
                { id: 5, address: 'machine5' },
                { id: 7, address: 'machine7' },
            ],
        }
        spyOn(service.servicesApi, 'getMachinesDirectory').and.returnValue(of(fakeResponse))
        service.getMachinesAddresses().subscribe((data) => {
            expect(data.size).toBe(2)
            expect(data.has('machine5')).toBeTrue()
            expect(data.has('machine7')).toBeTrue()
        })
        expect(service.servicesApi.getMachinesDirectory).toHaveBeenCalled()
    })

    it('should return app names', () => {
        const fakeResponse: any = {
            items: [
                { id: 100, name: 'lion' },
                { id: 110, name: 'frog' },
            ],
        }
        spyOn(service.servicesApi, 'getAppsDirectory').and.returnValue(of(fakeResponse))
        service.getAppsNames().subscribe((data) => {
            expect(data.size).toBe(2)
            expect(data.has('lion')).toBeTrue()
            expect(data.has('frog')).toBeTrue()
            expect(data.get('lion')).toBe(100)
            expect(data.get('frog')).toBe(110)
        })
        expect(service.servicesApi.getAppsDirectory).toHaveBeenCalled()
    })

    it('should return daemon configuration', fakeAsync(() => {
        const fakeResponse: any = {
            foo: 42,
            bar: {
                baz: 24,
            },
        }
        spyOn(service.servicesApi, 'getDaemonConfig').and.returnValue(of(fakeResponse))
        service.getDaemonConfiguration(1337).subscribe((data) => {
            expect(data).toEqual(
                jasmine.objectContaining({
                    foo: 42,
                    bar: {
                        baz: 24,
                    },
                })
            )
        })

        flush()
    }))

    it('should return error without broke pipe', fakeAsync(() => {
        const fakeError = new HttpErrorResponse({ status: 404 })
        spyOn(service.servicesApi, 'getDaemonConfig').and.returnValue(throwError(fakeError))
        service.getDaemonConfiguration(1337).subscribe((data) => {
            expect(data).toBeInstanceOf(HttpErrorResponse)
            expect((data as HttpErrorResponse).status).toBe(404)
        })
        flush()
    }))

    it('should reload daemon configuration', fakeAsync(() => {
        const fakeResponses: any = [
            {
                foo: 42,
            },
            {
                bar: 24,
            },
        ]
        const spy = spyOn(service.servicesApi, 'getDaemonConfig').and.returnValues(
            of(fakeResponses[0]),
            of(fakeResponses[1])
        )

        let subscribeCallCounts = 0
        service.getDaemonConfiguration(1337).subscribe((data) => {
            if (subscribeCallCounts === 0) {
                expect(data).toEqual(
                    jasmine.objectContaining({
                        foo: 42,
                    })
                )
            } else if (subscribeCallCounts === 1) {
                expect(data).toEqual(
                    jasmine.objectContaining({
                        bar: 24,
                    })
                )
            } else {
                fail()
            }
            subscribeCallCounts += 1
        })

        expect(spy).toHaveBeenCalledTimes(1)
        service.forceReloadDaemonConfiguration(1337)
        expect(spy).toHaveBeenCalledTimes(2)

        flush()
    }))
})
