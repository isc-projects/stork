import { ComponentFixture, TestBed } from '@angular/core/testing'

import { HostFormComponent } from './host-form.component'

describe('HostFormComponent', () => {
    let component: HostFormComponent
    let fixture: ComponentFixture<HostFormComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [HostFormComponent],
        }).compileComponents()
    })

    beforeEach(() => {
        fixture = TestBed.createComponent(HostFormComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
