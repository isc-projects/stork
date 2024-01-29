import { HttpClientTestingModule } from '@angular/common/http/testing'
import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'
import { ActivatedRoute, convertToParamMap, RouterModule } from '@angular/router'
import { By } from '@angular/platform-browser'
import { ServicesService } from '../backend'
import { LogViewPageComponent } from './log-view-page.component'
import { of } from 'rxjs'
import { PanelModule } from 'primeng/panel'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { ButtonModule } from 'primeng/button'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { SharedModule } from 'primeng/api'
import { EntityLinkComponent } from '../entity-link/entity-link.component'
import { RouterTestingModule } from '@angular/router/testing'

describe('LogViewPageComponent', () => {
    let component: LogViewPageComponent
    let fixture: ComponentFixture<LogViewPageComponent>

    beforeEach(waitForAsync(() => {
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
            imports: [
                HttpClientTestingModule,
                PanelModule,
                NoopAnimationsModule,
                ButtonModule,
                ProgressSpinnerModule,
                SharedModule,
                RouterModule,
                RouterTestingModule,
            ],
            declarations: [LogViewPageComponent, EntityLinkComponent],
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
