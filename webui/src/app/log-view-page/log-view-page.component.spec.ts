import { provideHttpClientTesting } from '@angular/common/http/testing'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { provideRouter } from '@angular/router'
import { By } from '@angular/platform-browser'
import { LogViewPageComponent } from './log-view-page.component'
import { provideNoopAnimations } from '@angular/platform-browser/animations'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'

describe('LogViewPageComponent', () => {
    let component: LogViewPageComponent
    let fixture: ComponentFixture<LogViewPageComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            providers: [
                provideNoopAnimations(),
                provideHttpClient(withInterceptorsFromDi()),
                provideHttpClientTesting(),
                provideRouter([]),
            ],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(LogViewPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should include daemon link', () => {
        component.loaded = true
        component.data = {
            logTargetOutput: '/tmp/xyz',
            daemonId: 15,
            daemonLabel: 'fantastic-daemon',
            contents: [],
        }
        fixture.detectChanges()
        const daemonLink = fixture.debugElement.query(By.css('#daemon-link'))
        const daemonLinkComponent = daemonLink.componentInstance
        expect(daemonLinkComponent).toBeDefined()
        expect(daemonLinkComponent.attrs.hasOwnProperty('daemonLabel')).toBeTrue()
        expect(daemonLinkComponent.attrs.daemonLabel).toEqual('fantastic-daemon')
        expect(daemonLinkComponent.attrs.daemonId).toEqual(15)
    })
})
