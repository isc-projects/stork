import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { LoginScreenComponent } from './login-screen.component'
import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { GeneralService, UsersService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { RouterModule, Router, ActivatedRoute } from '@angular/router'
import { MessageService } from 'primeng/api'
import { ProgressSpinnerModule } from 'primeng/progressspinner'

describe('LoginScreenComponent', () => {
    let component: LoginScreenComponent
    let fixture: ComponentFixture<LoginScreenComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [ReactiveFormsModule, FormsModule, RouterModule, HttpClientTestingModule, ProgressSpinnerModule],
            declarations: [LoginScreenComponent],
            providers: [
                GeneralService,
                UsersService,
                MessageService,
                {
                    provide: Router,
                    useValue: {},
                },
                {
                    provide: ActivatedRoute,
                    useValue: {
                        snapshot: { queryParams: {} },
                    },
                },
            ],
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
