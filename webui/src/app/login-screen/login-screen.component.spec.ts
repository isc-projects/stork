import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { LoginScreenComponent } from './login-screen.component'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { GeneralService, UsersService } from '../backend'
import { HttpClient, HttpHandler } from '@angular/common/http'
import { RouterModule, Router, ActivatedRoute } from '@angular/router'
import { MessageService } from 'primeng/api'

describe('LoginScreenComponent', () => {
    let component: LoginScreenComponent
    let fixture: ComponentFixture<LoginScreenComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [ReactiveFormsModule, FormsModule, RouterModule],
            declarations: [LoginScreenComponent],
            providers: [GeneralService, HttpClient, HttpHandler, UsersService, MessageService, {
                provide: Router,
                useValue: {}
            }, {
                provide: ActivatedRoute,
                useValue: {}
            }]
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
})
