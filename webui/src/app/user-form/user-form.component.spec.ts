import { ComponentFixture, fakeAsync, TestBed, tick } from '@angular/core/testing'

import { UserFormComponent } from './user-form.component'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { MessageService } from 'primeng/api'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { UserFormState } from '../forms/user-form'

describe('UserFormComponent', () => {
    let component: UserFormComponent
    let fixture: ComponentFixture<UserFormComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [UserFormComponent, NoopAnimationsModule],
            providers: [provideHttpClientTesting(), provideHttpClient(withInterceptorsFromDi()), MessageService],
        }).compileComponents()

        fixture = TestBed.createComponent(UserFormComponent)
        component = fixture.componentInstance
        component.formState = new UserFormState()
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should allow spaces in the password', fakeAsync(() => {
        // Initially the form should be invalid because it is empty.
        expect(component.formGroup).toBeTruthy()
        expect(component.formGroup.valid).toBeFalse()

        // Set valid data including a password containing spaces.
        component.formGroup.get('userLogin').setValue('frank')
        component.formGroup.get('userFirst').setValue('Frank')
        component.formGroup.get('userLast').setValue('Smith')
        component.formGroup.get('userGroup').setValue(1)
        component.formGroup.get('userPassword').setValue('password with spaces is cool')
        component.formGroup.get('userPassword2').setValue('password with spaces is cool')
        tick()
        fixture.detectChanges()

        // The form should be validated ok.
        expect(component.formGroup.valid).toBeTrue()
    }))
})
