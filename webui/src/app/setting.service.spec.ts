import { TestBed } from '@angular/core/testing'

import { SettingService } from './setting.service'
import { SettingsService, UsersService } from './backend'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { Router } from '@angular/router'
import { MessageService } from 'primeng/api'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

describe('SettingService', () => {
    beforeEach(() =>
        TestBed.configureTestingModule({
            imports: [],
            providers: [
                SettingsService,
                UsersService,
                MessageService,
                {
                    provide: Router,
                    useValue: {},
                },
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
            ],
        })
    )

    it('should be created', () => {
        const service: SettingService = TestBed.inject(SettingService)
        expect(service).toBeTruthy()
    })
})
