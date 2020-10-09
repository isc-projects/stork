import { HttpClient, HttpHandler } from '@angular/common/http'
import { async, ComponentFixture, TestBed } from '@angular/core/testing'
import { ActivatedRoute } from '@angular/router'
import { ServicesService } from '../backend'
import { LogViewPageComponent } from './log-view-page.component'

describe('LogViewPageComponent', () => {
    let component: LogViewPageComponent
    let fixture: ComponentFixture<LogViewPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            providers: [
                ServicesService, HttpClient, HttpHandler,
                {
                    provide: ActivatedRoute,
                    useValue: {},
                }
            ],
            declarations: [LogViewPageComponent],
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
})
