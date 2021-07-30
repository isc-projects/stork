import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { ActivatedRoute, convertToParamMap } from '@angular/router'
import { By } from '@angular/platform-browser'
import { ServicesService } from '../backend'
import { LogViewPageComponent } from './log-view-page.component'
import { of } from 'rxjs'

describe('LogViewPageComponent', () => {
    let component: LogViewPageComponent
    let fixture: ComponentFixture<LogViewPageComponent>

    beforeEach(
        waitForAsync(() => {
            TestBed.configureTestingModule({
                providers: [
                    ServicesService,
                    {
                        provide: ActivatedRoute,
                        useValue: {
                            paramMap: of(convertToParamMap({})),
                        },
                    },
                ],
                imports: [HttpClientTestingModule],
                declarations: [LogViewPageComponent],
            }).compileComponents()
        })
    )

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
        expect(appLink.properties.hasOwnProperty('attrs')).toBeTrue()
        expect(appLink.properties.attrs.hasOwnProperty('name')).toBeTrue()
        expect(appLink.properties.attrs.name).toEqual('fantastic-app')
    })
})
