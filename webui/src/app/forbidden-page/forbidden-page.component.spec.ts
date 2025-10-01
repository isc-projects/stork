import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { ForbiddenPageComponent } from './forbidden-page.component'
import { MessageModule } from 'primeng/message'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'

describe('ForbiddenPageComponent', () => {
    let component: ForbiddenPageComponent
    let fixture: ComponentFixture<ForbiddenPageComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [MessageModule, NoopAnimationsModule],
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
