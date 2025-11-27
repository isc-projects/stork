import { ComponentFixture, TestBed } from '@angular/core/testing'

import { Bind9ConfigPreviewComponent } from './bind9-config-preview.component'
import { MessageService } from 'primeng/api'
import { provideHttpClient, withInterceptorsFromDi } from '@angular/common/http'
import { CheckboxChangeEvent } from 'primeng/checkbox'
import { Bind9FormattedConfig } from '../backend'
import { provideHttpClientTesting } from '@angular/common/http/testing'

describe('Bind9ConfigPreviewComponent', () => {
    let component: Bind9ConfigPreviewComponent
    let fixture: ComponentFixture<Bind9ConfigPreviewComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            providers: [MessageService, provideHttpClientTesting(), provideHttpClient(withInterceptorsFromDi())],
        }).compileComponents()

        fixture = TestBed.createComponent(Bind9ConfigPreviewComponent)
        component = fixture.componentInstance

        // Set required inputs
        component.daemonId = 1
        component.fileType = 'config'

        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('it should handle the config refresh', () => {
        spyOn(component.bind9ConfigViewFeeder, 'updateConfig')
        component.handleConfigRefresh()
        expect(component.bind9ConfigViewFeeder.updateConfig).toHaveBeenCalledWith(false)
        component.showFullConfig = true
        component.handleConfigRefresh()
        expect(component.bind9ConfigViewFeeder.updateConfig).toHaveBeenCalledWith(true)
    })

    it('it should handle the full config toggle', () => {
        spyOn(component.bind9ConfigViewFeeder, 'updateConfig')
        component.handleFullConfigToggle({ checked: true } as CheckboxChangeEvent)
        expect(component.showFullConfig).toBeTrue()
        expect(component.bind9ConfigViewFeeder.updateConfig).toHaveBeenCalledWith(true)
    })

    it('it should handle the short config toggle', () => {
        spyOn(component.bind9ConfigViewFeeder, 'updateConfig')
        component.handleFullConfigToggle({ checked: false } as CheckboxChangeEvent)
        expect(component.showFullConfig).toBeFalse()
        expect(component.bind9ConfigViewFeeder.updateConfig).toHaveBeenCalledWith(false)
    })

    it('it should handle the config change', () => {
        const config: Bind9FormattedConfig = {
            files: [
                {
                    sourcePath: 'test.conf',
                    fileType: 'config',
                    contents: ['config;'],
                },
            ],
        }
        component.handleConfigChange(config)
        expect(component.config).toBe(config)
    })

    it('it should handle the visible change to true', () => {
        spyOn(component.visibleChange, 'emit')
        spyOn(component.bind9ConfigViewFeeder, 'cancelUpdateConfig')
        component.handleVisibleChange(true)
        expect(component.visible).toBeTrue()
        expect(component.visibleChange.emit).toHaveBeenCalledWith(true)
        expect(component.bind9ConfigViewFeeder.cancelUpdateConfig).not.toHaveBeenCalled()
    })

    it('it should handle the visible change to false', () => {
        spyOn(component.visibleChange, 'emit')
        spyOn(component.bind9ConfigViewFeeder, 'cancelUpdateConfig')
        component.handleVisibleChange(false)
        expect(component.visible).toBeFalse()
        expect(component.visibleChange.emit).toHaveBeenCalledWith(false)
        expect(component.bind9ConfigViewFeeder.cancelUpdateConfig).toHaveBeenCalled()
    })
})
