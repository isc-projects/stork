import { TestBed } from '@angular/core/testing'

import { SettingService } from './setting.service'
import { SettingsService } from './backend'
import { HttpClient, HttpHandler } from '@angular/common/http'

describe('SettingService', () => {
    beforeEach(() =>
        TestBed.configureTestingModule({
            providers: [SettingsService, HttpClient, HttpHandler],
        })
    )

    it('should be created', () => {
        const service: SettingService = TestBed.inject(SettingService)
        expect(service).toBeTruthy()
    })
})
