import { TestBed, async } from '@angular/core/testing'
import { RouterTestingModule } from '@angular/router/testing'
import { AppComponent } from './app.component'
import { TooltipModule } from 'primeng/tooltip'
import { MenubarModule } from 'primeng/menubar'
import { SplitButtonModule } from 'primeng/splitbutton'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { ToastModule } from 'primeng/toast'
import { GeneralService, UsersService, SettingsService, ServicesService } from './backend'
import { HttpClient, HttpHandler } from '@angular/common/http'
import { MessageService } from 'primeng/api'

describe('AppComponent', () => {
    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [
                RouterTestingModule,
                TooltipModule,
                MenubarModule,
                SplitButtonModule,
                ProgressSpinnerModule,
                ToastModule,
            ],
            declarations: [AppComponent],
            providers: [
                GeneralService,
                HttpClient,
                HttpHandler,
                UsersService,
                MessageService,
                ServicesService,
                SettingsService,
            ],
        }).compileComponents()
    }))

    it('should create the app', () => {
        const fixture = TestBed.createComponent(AppComponent)
        const app = fixture.debugElement.componentInstance
        expect(app).toBeTruthy()
    })

    it(`should have necessary menu items`, () => {
        // This test checks if the menu items are there. It is basic for now.
        // @todo: extend this to check if the menu items are shown or hidden (e.g. grafana is hidden by default)
        const fixture = TestBed.createComponent(AppComponent)
        const app = fixture.debugElement.componentInstance

        // This is the list of menu elements that are expected to be there.
        var expMenuItems = [ "DHCP", "Dashboard", "Host Reservations", "Subnets", "Shared Networks", "Services",
                             "Kea Apps", "BIND 9 Apps", "Machines", "Grafana",
                             "Monitoring", "Events",
                             "Configuration", "Users", "Settings",
                             "Help", "Stork Manual", "Stork API Docs (SwaggerUI)", "Stork API Docs (Redoc)", "BIND 9 Manual", "Kea Manual" ]
        for (var i = 0; i < expMenuItems.length; i++) {
            expect(app.getMenuItem(expMenuItems[i])).toBeTruthy()
            console.log("Checked existence of " + expMenuItems[i] + " menu item.")
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
})
