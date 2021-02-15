import { TestBed } from '@angular/core/testing'
import { ServerDataService } from './server-data.service'
import { HttpClient, HttpHandler } from '@angular/common/http'
import { Router } from '@angular/router'
import { ServicesService, UsersService } from './backend'
import { MessageService } from 'primeng/api'
import { of } from 'rxjs'

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
})
