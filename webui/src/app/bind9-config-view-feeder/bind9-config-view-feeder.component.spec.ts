import { ComponentFixture, fakeAsync, TestBed, tick } from '@angular/core/testing'

import { Bind9ConfigViewFeederComponent } from './bind9-config-view-feeder.component'
import { MessageService } from 'primeng/api'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { Bind9FormattedConfig, ServicesService } from '../backend'
import { of, throwError } from 'rxjs'

describe('Bind9ConfigViewFeederComponent', () => {
    let component: Bind9ConfigViewFeederComponent
    let fixture: ComponentFixture<Bind9ConfigViewFeederComponent>
    let servicesServiceSpy: jasmine.SpyObj<ServicesService>
    let messageServiceSpy: jasmine.SpyObj<MessageService>

    const mockShortConfigResponse: Bind9FormattedConfig = {
        files: [
            {
                sourcePath: 'named.conf',
                fileType: 'config',
                contents: ['config;'],
            },
        ],
    }

    const mockLongConfigResponse: Bind9FormattedConfig = {
        files: [
            {
                sourcePath: 'named.conf',
                fileType: 'config',
                contents: ['config;', 'view;'],
            },
        ],
    }

    const mockRndcKeyConfigResponse: Bind9FormattedConfig = {
        files: [
            {
                sourcePath: 'rndc.key',
                fileType: 'rndc-key',
                contents: ['rndc-key;'],
            },
        ],
    }

    beforeEach(async () => {
        const servicesSpy = jasmine.createSpyObj('ServicesService', ['getBind9FormattedConfig'])
        const messageSpy = jasmine.createSpyObj('MessageService', ['add'])

        await TestBed.configureTestingModule({
            imports: [Bind9ConfigViewFeederComponent],
            providers: [
                MessageService,
                provideHttpClient(withInterceptorsFromDi()),
                { provide: ServicesService, useValue: servicesSpy },
                { provide: MessageService, useValue: messageSpy },
            ],
        }).compileComponents()

        servicesServiceSpy = TestBed.inject(ServicesService) as jasmine.SpyObj<ServicesService>
        messageServiceSpy = TestBed.inject(MessageService) as jasmine.SpyObj<MessageService>

        fixture = TestBed.createComponent(Bind9ConfigViewFeederComponent)
        component = fixture.componentInstance

        // Set required inputs
        component.daemonId = 1
        component.fileType = 'config'
        component.active = false
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
        expect((component as any)['_loaded']).toBeFalse()
        expect(component.loading).toBeFalse()
        expect(component.config).toBeNull()
    })

    it('should fetch the configuration when the component is initialized', fakeAsync(() => {
        servicesServiceSpy.getBind9FormattedConfig.and.returnValue(of(mockShortConfigResponse as any))
        spyOn(component.configChange, 'emit')
        component.active = true
        tick()
        expect(servicesServiceSpy.getBind9FormattedConfig).toHaveBeenCalledWith(
            component.daemonId,
            ['config'],
            ['config']
        )
        expect(messageServiceSpy.add).not.toHaveBeenCalled()
        expect(component.configChange.emit).toHaveBeenCalled()
        expect((component as any)['_loaded']).toBeTrue()
        expect(component.loading).toBeFalse()
    }))

    it('should fetch the long configuration when requested', fakeAsync(() => {
        servicesServiceSpy.getBind9FormattedConfig.and.returnValue(of(mockLongConfigResponse as any))
        spyOn(component.configChange, 'emit')
        component.updateConfig(true)
        tick()
        expect(servicesServiceSpy.getBind9FormattedConfig).toHaveBeenCalledWith(component.daemonId, null, ['config'])
        expect(messageServiceSpy.add).not.toHaveBeenCalled()
        expect(component.configChange.emit).toHaveBeenCalled()
        expect((component as any)['_loaded']).toBeTrue()
        expect(component.loading).toBeFalse()
    }))

    it('should fetch the rndc key configuration when requested', fakeAsync(() => {
        servicesServiceSpy.getBind9FormattedConfig.and.returnValue(of(mockRndcKeyConfigResponse as any))
        spyOn(component.configChange, 'emit')
        component.fileType = 'rndc-key'
        component.updateConfig(true)
        tick()
        expect(servicesServiceSpy.getBind9FormattedConfig).toHaveBeenCalledWith(component.daemonId, null, ['rndc-key'])
        expect(messageServiceSpy.add).not.toHaveBeenCalled()
        expect(component.configChange.emit).toHaveBeenCalled()
        expect((component as any)['_loaded']).toBeTrue()
        expect(component.loading).toBeFalse()
    }))

    it('should handle API errors', fakeAsync(() => {
        servicesServiceSpy.getBind9FormattedConfig.and.returnValue(throwError(() => new Error('API error')))
        spyOn(component.configChange, 'emit')
        component.updateConfig(true)
        tick()
        expect(servicesServiceSpy.getBind9FormattedConfig).toHaveBeenCalledWith(component.daemonId, null, ['config'])
        expect(messageServiceSpy.add).toHaveBeenCalledWith({
            severity: 'error',
            summary: 'Error getting BIND 9 configuration',
            detail: 'API error',
            life: 10000,
        })
        expect(component.configChange.emit).not.toHaveBeenCalled()
        expect((component as any)['_loaded']).toBeFalse()
        expect(component.loading).toBeFalse()
    }))
})
