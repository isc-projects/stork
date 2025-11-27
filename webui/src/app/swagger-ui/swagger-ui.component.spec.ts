import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { SwaggerUiComponent } from './swagger-ui.component'

describe('SwaggerUiComponent', () => {
    let component: SwaggerUiComponent
    let fixture: ComponentFixture<SwaggerUiComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({}).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(SwaggerUiComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
