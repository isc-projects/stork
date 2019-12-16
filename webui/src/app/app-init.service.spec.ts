import { TestBed } from '@angular/core/testing'

import { AppInitService } from './app-init.service'

describe('AppInitService', () => {
    beforeEach(() => TestBed.configureTestingModule({}))

    it('should be created', () => {
        const service: AppInitService = TestBed.get(AppInitService)
        expect(service).toBeTruthy()
    })
})
