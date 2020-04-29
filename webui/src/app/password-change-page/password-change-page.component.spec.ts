import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { PasswordChangePageComponent } from './password-change-page.component'
import { HttpClient, HttpHandler } from '@angular/common/http'
import { Router } from '@angular/router'
import { FormBuilder } from '@angular/forms'
import { UsersService } from '../backend'
import { MessageService } from 'primeng/api'

describe('PasswordChangePageComponent', () => {
    let component: PasswordChangePageComponent
    let fixture: ComponentFixture<PasswordChangePageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [ FormBuilder, UsersService, HttpClient, HttpHandler, MessageService, {
                provide: Router, useValue: {}
            }],
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
