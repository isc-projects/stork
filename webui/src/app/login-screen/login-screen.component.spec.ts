import { ComponentFixture, fakeAsync, TestBed, tick, waitForAsync } from '@angular/core/testing'

import { LoginScreenComponent } from './login-screen.component'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { GeneralService, UsersService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { SelectButtonModule } from 'primeng/selectbutton'
import { ButtonModule } from 'primeng/button'
import { of } from 'rxjs'
import { By } from '@angular/platform-browser'
import { RouterModule } from '@angular/router'
import { MessagesModule } from 'primeng/messages'

describe('LoginScreenComponent', () => {
    let component: LoginScreenComponent
    let fixture: ComponentFixture<LoginScreenComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [
                ReactiveFormsModule,
                FormsModule,
                RouterModule.forRoot([]),
                HttpClientTestingModule,
                ProgressSpinnerModule,
                SelectButtonModule,
                ButtonModule,
                MessagesModule,
            ],
            declarations: [LoginScreenComponent],
            providers: [GeneralService, UsersService, MessageService],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(LoginScreenComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display welcome message', fakeAsync(() => {
        spyOn(component.http, 'get').and.returnValue(of('This is a welcome message'))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        const welcomeMessage = fixture.debugElement.query(By.css('.login-screen__welcome'))
        expect(welcomeMessage).toBeTruthy()
        expect(welcomeMessage.nativeElement.innerText).toContain('This is a welcome message')
    }))

    it('should not display bloated welcome message', fakeAsync(() => {
        spyOn(component.http, 'get').and.returnValue(of('a'.repeat(2049)))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        const welcomeMessage = fixture.debugElement.query(By.css('.login-screen__welcome'))
        expect(welcomeMessage).toBeFalsy()
    }))
})
