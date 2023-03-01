import { ComponentFixture, TestBed } from '@angular/core/testing'

import { PasswordChangeFormComponent } from './password-change-form.component'
import { ReactiveFormsModule, UntypedFormBuilder } from '@angular/forms'
import { UsersService } from '../backend'
import { MessageService } from 'primeng/api'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { PasswordModule } from 'primeng/password'
import { PanelModule } from 'primeng/panel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'

describe('PasswordChangeFormComponent', () => {
    let component: PasswordChangeFormComponent
    let fixture: ComponentFixture<PasswordChangeFormComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [UntypedFormBuilder, UsersService, MessageService],
            imports: [HttpClientTestingModule, ReactiveFormsModule, PasswordModule, PanelModule, NoopAnimationsModule],
            declarations: [PasswordChangeFormComponent],
        }).compileComponents()

        fixture = TestBed.createComponent(PasswordChangeFormComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
