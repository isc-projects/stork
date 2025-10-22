import { ComponentFixture, TestBed } from '@angular/core/testing'

import { Bind9DaemonControlsComponent } from './bind9-daemon-controls.component'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { MessageService } from 'primeng/api'

describe('Bind9DaemonControlsComponent', () => {
    let component: Bind9DaemonControlsComponent
    let fixture: ComponentFixture<Bind9DaemonControlsComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [Bind9DaemonControlsComponent],
            providers: [MessageService, provideHttpClient(withInterceptorsFromDi())],
        }).compileComponents()

        fixture = TestBed.createComponent(Bind9DaemonControlsComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('it should set the dialog visible to true', () => {
        component.setDialogVisible('config')
        expect(component.dialogVisible['config']).toBeTrue()
        expect(component.dialogVisible['rndc-key']).toBeFalse()
    })

    it('it should set the dialog visible to false', () => {
        component.setDialogVisible('rndc-key')
        expect(component.dialogVisible['config']).toBeFalse()
        expect(component.dialogVisible['rndc-key']).toBeTrue()
    })
})
