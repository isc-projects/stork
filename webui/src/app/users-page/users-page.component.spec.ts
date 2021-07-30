import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { UsersPageComponent } from './users-page.component'
import { ActivatedRoute, Router } from '@angular/router'
import { FormBuilder } from '@angular/forms'
import { ServicesService, UsersService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { of } from 'rxjs'

class MockParamMap {
    get(name: string): string | null {
        return null
    }
}

describe('UsersPageComponent', () => {
    let component: UsersPageComponent
    let fixture: ComponentFixture<UsersPageComponent>

    beforeEach(
        waitForAsync(() => {
            TestBed.configureTestingModule({
                imports: [HttpClientTestingModule],
                declarations: [UsersPageComponent],
                providers: [
                    FormBuilder,
                    UsersService,
                    ServicesService,
                    MessageService,
                    {
                        provide: ActivatedRoute,
                        useValue: {
                            paramMap: of(new MockParamMap()),
                        },
                    },
                    {
                        provide: Router,
                        useValue: {},
                    },
                ],
            }).compileComponents()
        })
    )

    beforeEach(() => {
        fixture = TestBed.createComponent(UsersPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
