import { ComponentFixture, TestBed } from '@angular/core/testing'

import { NotFoundPageComponent } from './not-found-page.component'
import { MessageModule } from 'primeng/message'
import { MessagesModule } from 'primeng/messages'

describe('NotFoundPageComponent', () => {
    let component: NotFoundPageComponent
    let fixture: ComponentFixture<NotFoundPageComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [MessageModule, MessagesModule],
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
