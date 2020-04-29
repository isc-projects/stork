import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { ProfilePageComponent } from './profile-page.component'
import { HttpClient, HttpHandler } from '@angular/common/http'
import { UsersService } from '../backend'
import { Router } from '@angular/router'
import { MessageService } from 'primeng/api'
import { AuthService } from '../auth.service'

describe('ProfilePageComponent', () => {
    let component: ProfilePageComponent
    let fixture: ComponentFixture<ProfilePageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [MessageService, UsersService, HttpClient, HttpHandler, { provide: Router, useValue: {} }],
            declarations: [ProfilePageComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(ProfilePageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
