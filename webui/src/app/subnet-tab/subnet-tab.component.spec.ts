import { ComponentFixture, TestBed } from '@angular/core/testing'

import { SubnetTabComponent } from './subnet-tab.component'

describe('SubnetTabComponent', () => {
    let component: SubnetTabComponent
    let fixture: ComponentFixture<SubnetTabComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            declarations: [SubnetTabComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(SubnetTabComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
