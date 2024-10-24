import { By } from '@angular/platform-browser'
import { ComponentFixture, fakeAsync, TestBed, tick, waitForAsync } from '@angular/core/testing'
import { differentPasswords, UsersPageComponent } from './users-page.component'
import { ActivatedRoute, convertToParamMap, ParamMap, RouterModule } from '@angular/router'
import { FormControl, FormGroup, FormsModule, UntypedFormBuilder } from '@angular/forms'
import { ServicesService, UsersService } from '../backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ConfirmationService, MessageService, SharedModule } from 'primeng/api'
import { of, Subject } from 'rxjs'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { TabMenuModule } from 'primeng/tabmenu'
import { MenuModule } from 'primeng/menu'
import { TableModule } from 'primeng/table'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { ReactiveFormsModule } from '@angular/forms'
import { AuthService } from '../auth.service'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { MockParamMap } from '../utils'
import { TagModule } from 'primeng/tag'
import { PanelModule } from 'primeng/panel'
import { DropdownModule } from 'primeng/dropdown'
import { PasswordModule } from 'primeng/password'
import { CheckboxModule } from 'primeng/checkbox'

describe('UsersPageComponent', () => {
    let component: UsersPageComponent
    let fixture: ComponentFixture<UsersPageComponent>
    let usersApi: UsersService
    let confirmService: ConfirmationService
    let paramMapValue: Subject<ParamMap> = new Subject()

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            imports: [
                HttpClientTestingModule,
                TabMenuModule,
                MenuModule,
                TableModule,
                BreadcrumbModule,
                OverlayPanelModule,
                NoopAnimationsModule,
                RouterModule,
                ReactiveFormsModule,
                ConfirmDialogModule,
                SharedModule,
                TagModule,
                RouterModule.forRoot([
                    { path: 'users/1', component: UsersPageComponent },
                    { path: 'users/new', component: UsersPageComponent },
                ]),
                PanelModule,
                DropdownModule,
                PasswordModule,
                FormsModule,
                CheckboxModule,
            ],
            declarations: [UsersPageComponent, BreadcrumbsComponent, HelpTipComponent, PlaceholderPipe],
            providers: [
                UntypedFormBuilder,
                UsersService,
                ServicesService,
                MessageService,
                ConfirmationService,
                {
                    provide: ActivatedRoute,
                    useValue: {
                        paramMap: of(new MockParamMap()),
                    },
                },
                {
                    provide: AuthService,
                    useValue: {
                        currentUser: of({}),
                        currentUserValue: { id: 1 },
                    },
                },
                {
                    provide: ActivatedRoute,
                    useValue: {
                        snapshot: { queryParamMap: new MockParamMap() },
                        queryParamMap: of(new MockParamMap()),
                        paramMap: paramMapValue.asObservable(),
                    },
                },
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(UsersPageComponent)
        component = fixture.componentInstance
        usersApi = fixture.debugElement.injector.get(UsersService)
        confirmService = fixture.debugElement.injector.get(ConfirmationService)
        paramMapValue.next(convertToParamMap({ id: 'list' }))
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should have breadcrumbs', () => {
        const breadcrumbsElement = fixture.debugElement.query(By.directive(BreadcrumbsComponent))
        expect(breadcrumbsElement).not.toBeNull()
        const breadcrumbsComponent = breadcrumbsElement.componentInstance as BreadcrumbsComponent
        expect(breadcrumbsComponent).not.toBeNull()
        expect(breadcrumbsComponent.items).toHaveSize(2)
        expect(breadcrumbsComponent.items[0].label).toEqual('Configuration')
        expect(breadcrumbsComponent.items[1].label).toEqual('Users')
    })

    it('should delete user when pressing delete button', async () => {
        spyOn(confirmService, 'confirm').and.callThrough()
        spyOn(usersApi, 'deleteUser')
        component.ngOnInit()
        await fixture.whenStable()
        // Only users tab should be populated with no users
        expect(component.openedTabs.length).toBe(1)
        expect(component.users.length).toBe(0)
        // Populate the users list with our data
        component.users = [
            {
                id: 1,
                lastname: 'user_one',
                login: '',
                name: 'user_one',
                email: 'user_one@isc.org',
                groups: [],
                changePassword: false,
            },
            {
                id: 2,
                lastname: 'user_two',
                login: '',
                name: 'user_two',
                email: 'user_two@isc.org',
                groups: [],
                changePassword: true,
            },
        ]
        component.totalUsers = 2
        fixture.detectChanges()
        // Check that the list is updated
        expect(component.users.length).toBe(2)
        expect(component.openedTabs.length).toBe(1)
        // Trigger the event to open the user tab
        paramMapValue.next(convertToParamMap({ id: '1' }))
        // Check that there are 2 tabs
        expect(component.openedTabs.length).toBe(2)
        // Check that the current tab is the user tab
        expect(component.existingUserTab).toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()
        await fixture.whenRenderingDone()
        // Detect the delete button and press it
        const deleteBtn = fixture.debugElement.query(By.css('[label=Delete]'))
        expect(deleteBtn).toBeTruthy()
        // Simulate clicking on the button and make sure that the confirm dialog
        // has been displayed.
        deleteBtn.nativeElement.click()
        expect(confirmService.confirm).toHaveBeenCalled()
        await fixture.whenStable()
        fixture.detectChanges()
        await fixture.whenRenderingDone()
        // Detect the yes button and press it
        const yesBtn = fixture.debugElement.query(By.css('.p-confirm-dialog-accept'))
        yesBtn.nativeElement.click()
        // Check that the deleteUser function has been called
        expect(usersApi.deleteUser).toHaveBeenCalled()
    })

    it('should allow spaces in the password', fakeAsync(() => {
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        // Open the new user tab.
        paramMapValue.next(convertToParamMap({ id: 'new' }))
        tick()
        fixture.detectChanges()

        // Initially the form should be invalid because it is empty.
        expect(component.userForm).toBeTruthy()
        expect(component.userForm.valid).toBeFalse()

        // Set valid data including a password containing spaces.
        component.userForm.get('userLogin').setValue('frank')
        component.userForm.get('userFirst').setValue('Frank')
        component.userForm.get('userLast').setValue('Smith')
        component.userForm.get('userGroup').setValue(1)
        component.userForm.get('userPassword').setValue('password with spaces is cool')
        component.userForm.get('userPassword2').setValue('password with spaces is cool')
        tick()
        fixture.detectChanges()

        // The form should be validated ok.
        expect(component.userForm.valid).toBeTrue()
    }))

    it('should verify if the passwords are the same', () => {
        const formGroup = new FormGroup({
            oldPassword: new FormControl('password'),
            newPassword: new FormControl('password'),
        })

        const validator = differentPasswords('oldPassword', 'newPassword')
        expect(validator(formGroup)).toEqual({ samePasswords: true })
    })

    it('should verify if the passwords are not the same', () => {
        const formGroup = new FormGroup({
            oldPassword: new FormControl('password'),
            newPassword: new FormControl('another-password'),
        })

        const validator = differentPasswords('oldPassword', 'newPassword')
        expect(validator(formGroup)).toBeNull()
    })
})
