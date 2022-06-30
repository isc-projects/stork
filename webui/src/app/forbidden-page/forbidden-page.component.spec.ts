import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { MessageModule } from 'primeng/message'

import { ForbiddenPageComponent } from './forbidden-page.component'

describe('ForbiddenPageComponent', () => {
    let component: ForbiddenPageComponent
    let fixture: ComponentFixture<ForbiddenPageComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [MessageModule],
            declarations: [ForbiddenPageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(ForbiddenPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
