import { async, ComponentFixture, TestBed } from '@angular/core/testing'

import { FormsModule, ReactiveFormsModule } from '@angular/forms'
import { BrowserAnimationsModule } from '@angular/platform-browser/animations'
import { FieldsetModule } from 'primeng/fieldset'
import { MessageService } from 'primeng/api'
import { HttpClient, HttpHandler } from '@angular/common/http'

import { SettingsPageComponent } from './settings-page.component'
import { SettingsService } from '../backend/api/api'

describe('SettingsPageComponent', () => {
    let component: SettingsPageComponent
    let fixture: ComponentFixture<SettingsPageComponent>

    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [FormsModule, ReactiveFormsModule, BrowserAnimationsModule, FieldsetModule],
            declarations: [SettingsPageComponent],
            providers: [SettingsService, MessageService, HttpClient, HttpHandler],
        }).compileComponents()
    }))

    beforeEach(() => {
        fixture = TestBed.createComponent(SettingsPageComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })
})
