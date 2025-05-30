import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { ForbiddenPageComponent } from './forbidden-page.component'
import { MessagesModule } from 'primeng/messages'

describe('ForbiddenPageComponent', () => {
    let component: ForbiddenPageComponent
    let fixture: ComponentFixture<ForbiddenPageComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [MessagesModule],
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
