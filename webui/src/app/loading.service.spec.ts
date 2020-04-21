import { TestBed } from '@angular/core/testing'

import { LoadingService } from './loading.service'

describe('LoadingService', () => {
    beforeEach(() => TestBed.configureTestingModule({}))

    it('should be created', () => {
        const service: LoadingService = TestBed.inject(LoadingService)
        expect(service).toBeTruthy()
    })
})
