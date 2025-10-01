import { ComponentFixture, TestBed } from '@angular/core/testing'

import { NotFoundPageComponent } from './not-found-page.component'
import { MessageModule } from 'primeng/message'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'

describe('NotFoundPageComponent', () => {
    let component: NotFoundPageComponent
    let fixture: ComponentFixture<NotFoundPageComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [MessageModule, NoopAnimationsModule],
            declarations: [NotFoundPageComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(NotFoundPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
