import { TestBed } from '@angular/core/testing'

import { SettingService } from './setting.service'
import { SettingsService, UsersService } from './backend'
import { HttpClient, HttpHandler } from '@angular/common/http'
import { ActivatedRoute, Router } from '@angular/router'
import { MessageService } from 'primeng/api'

describe('SettingService', () => {
    beforeEach(() =>
        TestBed.configureTestingModule({
            providers: [
                SettingsService,
                UsersService,
                MessageService,
                HttpClient,
                HttpHandler,
                {
                    provide: Router,
                    useValue: {},
                },
            ],
        })
    )

    it('should be created', () => {
        const service: SettingService = TestBed.inject(SettingService)
        expect(service).toBeTruthy()
    })
})
