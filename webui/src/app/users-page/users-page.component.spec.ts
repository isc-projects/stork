import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { differentPasswords, UsersPageComponent } from './users-page.component'
import { ActivatedRoute, convertToParamMap, ParamMap, RouterModule } from '@angular/router'
import { FormControl, FormGroup, FormsModule, UntypedFormBuilder } from '@angular/forms'
import { ServicesService, UsersService } from '../backend'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { ConfirmationService, MessageService, SharedModule } from 'primeng/api'
import { of, Subject } from 'rxjs'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { MenuModule } from 'primeng/menu'
import { TableModule } from 'primeng/table'
import { BreadcrumbModule } from 'primeng/breadcrumb'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { PopoverModule } from 'primeng/popover'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { ReactiveFormsModule } from '@angular/forms'
import { AuthService } from '../auth.service'
import { ConfirmDialogModule } from 'primeng/confirmdialog'
import { PlaceholderPipe } from '../pipes/placeholder.pipe'
import { MockParamMap } from '../utils'
import { TagModule } from 'primeng/tag'
import { PanelModule } from 'primeng/panel'
import { SelectModule } from 'primeng/select'
import { PasswordModule } from 'primeng/password'
import { CheckboxModule } from 'primeng/checkbox'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { ManagedAccessDirective } from '../managed-access.directive'
import { TabViewComponent } from '../tab-view/tab-view.component'
import { ButtonModule } from 'primeng/button'
import { IconFieldModule } from 'primeng/iconfield'
import { InputIconModule } from 'primeng/inputicon'

describe('UsersPageComponent', () => {
    let component: UsersPageComponent
    let fixture: ComponentFixture<UsersPageComponent>
    let usersApi: UsersService
    let confirmService: ConfirmationService
    let paramMapValue: Subject<ParamMap> = new Subject()

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            declarations: [UsersPageComponent, BreadcrumbsComponent, HelpTipComponent, PlaceholderPipe],
            imports: [
                MenuModule,
                TableModule,
                BreadcrumbModule,
                PopoverModule,
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
                SelectModule,
                PasswordModule,
                FormsModule,
                CheckboxModule,
                ManagedAccessDirective,
                TabViewComponent,
                ButtonModule,
                IconFieldModule,
                InputIconModule,
            ],
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
                        currentUser$: of({}),
                        currentUserValue: { id: 1 },
                        hasPrivilege: () => true,
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
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
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
        // expect(component.openedTabs.length).toBe(1)
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
                authenticationMethodId: 'internal',
            },
            {
                id: 2,
                lastname: 'user_two',
                login: '',
                name: 'user_two',
                email: 'user_two@isc.org',
                groups: [],
                changePassword: true,
                authenticationMethodId: 'internal',
            },
        ]
        component.totalUsers = 2
        fixture.detectChanges()
        // Check that the list is updated
        expect(component.users.length).toBe(2)
        // expect(component.openedTabs.length).toBe(1)
        // Trigger the event to open the user tab
        paramMapValue.next(convertToParamMap({ id: '1' }))
        // // Check that the current tab is the user tab
        // expect(component.existingUserTab).toBeTrue()
        await fixture.whenStable()
        fixture.detectChanges()
        await fixture.whenRenderingDone()
        // Check that there are 2 tabs
        expect(component.tabView().openTabs.length).toBe(2)
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
