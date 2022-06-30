import { ComponentFixture, TestBed, waitForAsync } from '@angular/core/testing'

import { AppDaemonsStatusComponent } from './app-daemons-status.component'

class Daemon {}

class Details {
    daemons: Daemon[]
}

class Machine {
    id = 1
}

class App {
    id = 1
    machine = new Machine()
    details = new Details()
}

describe('AppDaemonsStatusComponent', () => {
    let component: AppDaemonsStatusComponent
    let fixture: ComponentFixture<AppDaemonsStatusComponent>

    beforeEach(waitForAsync(() => {
        TestBed.configureTestingModule({
            declarations: [AppDaemonsStatusComponent],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(AppDaemonsStatusComponent)
        component = fixture.componentInstance
        component.app = new App()
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
