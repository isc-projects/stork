import { ComponentFixture, TestBed } from '@angular/core/testing'

import { VersionStatusComponent } from './version-status.component'

describe('VersionStatusComponent', () => {
    let component: VersionStatusComponent
    let fixture: ComponentFixture<VersionStatusComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [VersionStatusComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(VersionStatusComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
