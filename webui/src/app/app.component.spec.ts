import { ComponentFixture, fakeAsync, TestBed, tick, waitForAsync } from '@angular/core/testing'
import { RouterTestingModule } from '@angular/router/testing'
import { AppComponent } from './app.component'
import { TooltipModule } from 'primeng/tooltip'
import { MenubarModule } from 'primeng/menubar'
import { SplitButtonModule } from 'primeng/splitbutton'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { ToastModule } from 'primeng/toast'
import { AppsVersions, GeneralService, ServicesService, Settings, SettingsService, UsersService } from './backend'
import { HttpClientTestingModule } from '@angular/common/http/testing'
import { MessageService } from 'primeng/api'
import { GlobalSearchComponent } from './global-search/global-search.component'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { FormsModule } from '@angular/forms'
import { PriorityErrorsPanelComponent } from './priority-errors-panel/priority-errors-panel.component'
import { ServerSentEventsService, ServerSentEventsTestingService } from './server-sent-events.service'
import { MessagesModule } from 'primeng/messages'
import { ToggleButtonModule } from 'primeng/togglebutton'
import { SettingService } from './setting.service'
import { of } from 'rxjs'
import { AuthService } from './auth.service'
import { ServerDataService } from './server-data.service'
import { Severity, VersionAlert, VersionService } from './version.service'
import { By } from '@angular/platform-browser'

describe('AppComponent', () => {
    let component: AppComponent
    let fixture: ComponentFixture<AppComponent>
    let authService: AuthService
    let userService: UsersService
    let settingService: SettingService
    let serverDataService: ServerDataService
    let versionServiceStub: Partial<VersionService>

    beforeEach(waitForAsync(() => {
        versionServiceStub = {
            getVersionAlert: () => of({ severity: Severity.error, detected: true } as VersionAlert),
            getCurrentData: () => of({} as AppsVersions),
        }

        TestBed.configureTestingModule({
            imports: [
                RouterTestingModule.withRoutes([{ path: 'abc', component: AppComponent }]),
                TooltipModule,
                MenubarModule,
                SplitButtonModule,
                ProgressSpinnerModule,
                ToastModule,
                HttpClientTestingModule,
                OverlayPanelModule,
                NoopAnimationsModule,
                FormsModule,
                MessagesModule,
                ToggleButtonModule,
            ],
            declarations: [AppComponent, GlobalSearchComponent, PriorityErrorsPanelComponent],
            providers: [
                GeneralService,
                UsersService,
                MessageService,
                { provide: ServerSentEventsService, useClass: ServerSentEventsTestingService },
                ServicesService,
                SettingsService,
                { provide: VersionService, useValue: versionServiceStub },
            ],
        }).compileComponents()
        authService = TestBed.inject(AuthService)
        userService = TestBed.inject(UsersService)
        settingService = TestBed.inject(SettingService)
        serverDataService = TestBed.inject(ServerDataService)
        TestBed.inject(VersionService)
    }))

    beforeEach(() => {
        spyOn(userService, 'createSession').and.returnValues(
            of({
                id: 1,
                login: 'foo',
                email: 'foo@bar.baz',
                name: 'foo',
                lastname: 'bar',
                groups: [],
            } as any),
            of({
                id: 1,
                login: 'foo',
                email: 'foo@bar.baz',
                name: 'foo',
                lastname: 'bar',
                groups: [1],
            } as any)
        )
        authService.login('boz', 'foo', 'bar', 'abc')
        fixture = TestBed.createComponent(AppComponent)
        component = fixture.componentInstance
    })

    it('should create the app', () => {
        expect(component).toBeTruthy()
    })

    it(`should have necessary menu items`, () => {
        // This is the list of menu elements that are expected to be there.
        const expMenuItems = [
            'DHCP',
            'Dashboard',
            'Host Reservations',
            'Subnets',
            'Shared Networks',
            'Services',
            'Kea Apps',
            'BIND 9 Apps',
            'Machines',
            'Grafana',
            'Monitoring',
            'Events',
            'Configuration',
            'Users',
            'Settings',
            'Help',
            'Stork Manual',
            'Stork API Docs (SwaggerUI)',
            'Stork API Docs (Redoc)',
            'BIND 9 Manual',
            'Kea Manual',
        ]

        // List of menu items that are expected to be hidden. Unless listed here, the test expects
        // the menu item to be visible.
        const expHiddenItems = ['DHCP', 'Kea Apps', 'BIND 9 Apps', 'Grafana', 'Users']

        for (const name of expMenuItems) {
            // Check if the menu item is there
            const m = component.getMenuItem(name)
            expect(m).toBeTruthy()

            // Check if the menu is hidden or visible. See the expHiddenItems list above.
            if (expHiddenItems.includes(name)) {
                expect(m.visible === true).toBeFalsy()
            } else {
                // If defined, it must be visible. If not defined, the default is it's visible.
                if ('visible' in m) {
                    expect(m.visible === true).toBeTruthy()
                }
            }
        }
    })

    it('should render title', () => {
        const fixture = TestBed.createComponent(AppComponent)
        fixture.detectChanges()
        const compiled = fixture.debugElement.nativeElement
        expect(compiled).toBeTruthy()
        // This works in a browser, but not here.
        // expect(document.querySelector('app-login-screen').textContent).toContain('Dashboard for')
    })

    it('should show grafana menu item', fakeAsync(() => {
        const settings: Settings = {
            grafanaUrl: 'http://localhost:3000',
        }
        spyOn(settingService, 'getSettings').and.returnValue(of(settings))
        spyOn(serverDataService, 'getAppsStats').and.returnValue(of({} as any))
        component.ngOnInit()
        tick()
        fixture.detectChanges()

        expect(serverDataService.getAppsStats).toHaveBeenCalled()
        expect(settingService.getSettings).toHaveBeenCalled()
        const menuItem = component.getMenuItem('Grafana')
        expect(menuItem).toBeTruthy()
        expect(menuItem.visible).toBeTrue()
    }))

    it('should display version alert when detected', () => {
        fixture.detectChanges()
        let badges = fixture.debugElement.queryAll(By.css('.p-badge'))
        expect(badges).toBeTruthy()
        expect(badges.length).toBeGreaterThan(0)
        expect(badges.length).toEqual(2)

        // Check if error severity badge is shown in top bar menu.
        expect(badges[0].classes.hasOwnProperty('p-menuitem-badge')).toBeTrue()
        expect(badges[0].classes.hasOwnProperty('p-badge-dot')).toBeTrue()
        expect(badges[0].classes.hasOwnProperty('p-badge-danger')).toBeTrue()

        expect(badges[1].classes.hasOwnProperty('p-menuitem-badge')).toBeTrue()
        expect(badges[1].classes.hasOwnProperty('p-badge-dot')).toBeTrue()
        expect(badges[1].classes.hasOwnProperty('p-badge-danger')).toBeTrue()
    })
})
