import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { PasswordChangePageComponent } from './password-change-page.component'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { Router } from '@angular/router'
import { FormBuilder } from '@angular/forms'
import { UsersService } from '../backend'
import { MessageService } from 'primeng/api'

describe('PasswordChangePageComponent', () => {
    let component: PasswordChangePageComponent
    let fixture: ComponentFixture<PasswordChangePageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [
                FormBuilder,
                UsersService,
                MessageService,
                {
                    provide: Router,
                    useValue: {},
                },
            ],
            imports: [HttpClientTestingModule],
            declarations: [PasswordChangePageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(PasswordChangePageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
