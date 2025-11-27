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

    it('should include app link', () => {
        component.loaded = true
        component.data = { logTargetOutput: '/tmp/xyz', machine: { id: 1 } }
        component.appName = 'fantastic-app'
        fixture.detectChanges()
        const appLink = fixture.debugElement.query(By.css('#app-link'))
        const appLinkComponent = appLink.componentInstance
        expect(appLinkComponent).toBeDefined()
        expect(appLinkComponent.attrs.hasOwnProperty('name')).toBeTrue()
        expect(appLinkComponent.attrs.name).toEqual('fantastic-app')
    })
})
