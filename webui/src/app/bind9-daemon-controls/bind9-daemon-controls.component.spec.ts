import { ComponentFixture, TestBed } from '@angular/core/testing'

import { Bind9DaemonControlsComponent } from './bind9-daemon-controls.component'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { MessageService } from 'primeng/api'
import { provideHttpClientTesting } from '@angular/common/http/testing'

describe('Bind9DaemonControlsComponent', () => {
    let component: Bind9DaemonControlsComponent
    let fixture: ComponentFixture<Bind9DaemonControlsComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [MessageService, provideHttpClientTesting(), provideHttpClient(withInterceptorsFromDi())],
        }).compileComponents()

        fixture = TestBed.createComponent(Bind9DaemonControlsComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('it should set config dialog visibility', () => {
        component.setDialogVisibility('config', true)
        expect(component.dialogVisible['config']).toBeTrue()
        component.setDialogVisibility('config', false)
        expect(component.dialogVisible['config']).toBeFalse()
    })

    it('it should set rndc dialog visibility', () => {
        component.setDialogVisibility('rndc-key', true)
        expect(component.dialogVisible['rndc-key']).toBeTrue()
        component.setDialogVisibility('rndc-key', false)
        expect(component.dialogVisible['rndc-key']).toBeFalse()
    })
})
