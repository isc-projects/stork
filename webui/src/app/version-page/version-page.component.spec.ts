import { ComponentFixture, TestBed } from '@angular/core/testing'

import { VersionPageComponent } from './version-page.component'

describe('VersionPageComponent', () => {
    let component: VersionPageComponent
    let fixture: ComponentFixture<VersionPageComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [VersionPageComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(VersionPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
