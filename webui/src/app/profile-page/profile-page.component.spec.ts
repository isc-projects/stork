import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { ProfilePageComponent } from './profile-page.component'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ServicesService, UsersService } from '../backend'
import { Router } from '@angular/router'
import { MessageService } from 'primeng/api'
import { AuthService } from '../auth.service'
import { of } from 'rxjs'

describe('ProfilePageComponent', () => {
    let component: ProfilePageComponent
    let fixture: ComponentFixture<ProfilePageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [
                MessageService,
                UsersService,
                ServicesService,
                { provide: Router, useValue: {} },
                {
                    provide: AuthService,
                    useValue: {
                        currentUser: of({}),
                    },
                },
            ],
            declarations: [ProfilePageComponent],
            imports: [HttpClientTestingModule],
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
