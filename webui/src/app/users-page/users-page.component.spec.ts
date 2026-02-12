import { By } from '@angular/platform-browser'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { UsersPageComponent } from './users-page.component'
import { ActivatedRoute, convertToParamMap, ParamMap, provideRouter } from '@angular/router'
import { FormControl, FormGroup, UntypedFormBuilder } from '@angular/forms'
import { UsersService } from '../backend'
import { provideHttpClientTesting } from '@angular/common/http/testing'
import { ConfirmationService, MessageService } from 'primeng/api'
import { of, Subject } from 'rxjs'
import { BreadcrumbsComponent } from '../breadcrumbs/breadcrumbs.component'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { AuthService } from '../auth.service'
import { MockParamMap } from '../utils'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

describe('UsersPageComponent', () => {
    let component: UsersPageComponent
    let fixture: ComponentFixture<UsersPageComponent>
    let usersApi: UsersService
    let confirmService: ConfirmationService
    let paramMapValue: Subject<ParamMap> = new Subject()

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                UntypedFormBuilder,
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
                provideNoopAnimations(),
                provideRouter([
                    { path: 'users/1', component: UsersPageComponent },
                    { path: 'users/new', component: UsersPageComponent },
                ]),
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

    xit('should delete user when pressing delete button', async () => {
        // TODO: this test should be moved away from Karma tests.
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
        const yesBtn = fixture.debugElement.query(By.css('.p-confirmdialog-accept-button'))
        yesBtn.nativeElement.click()
        // Check that the deleteUser function has been called
        expect(usersApi.deleteUser).toHaveBeenCalled()
    })
})

describe('UsersPageComponent without superadmin privileges', () => {
    let component: UsersPageComponent
    let fixture: ComponentFixture<UsersPageComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                MessageService,
                ConfirmationService,
                provideHttpClient(withInterceptorsFromDi()),
                provideRouter([]),
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(UsersPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should have enabled or disabled button in filtering toolbar according to privileges', () => {
        expect(component.toolbarButtons.length).toBeGreaterThan(0)
        // at first, it should be disabled
        expect(component.toolbarButtons[0].disabled).toBeTrue()
        // it should react on privilege change
        component.canCreateUser.set(true)
        fixture.detectChanges()
        expect(component.toolbarButtons[0].disabled).toBeFalse()
    })
})
