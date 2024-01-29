import { TestBed } from '@angular/core/testing'

import { SettingService } from './setting.service'
import { SettingsService, UsersService } from './backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { Router } from '@angular/router'
import { MessageService } from 'primeng/api'

describe('SettingService', () => {
    beforeEach(() =>
        TestBed.configureTestingModule({
            providers: [
                SettingsService,
                UsersService,
                MessageService,
                {
                    provide: Router,
                    useValue: {},
                },
            ],
            imports: [HttpClientTestingModule],
        })
    )

    it('should be created', () => {
        const service: SettingService = TestBed.inject(SettingService)
        expect(service).toBeTruthy()
    })
})
