import { TestBed, async } from '@angular/core/testing'
import { RouterTestingModule } from '@angular/router/testing'
import { AppComponent } from './app.component'
import { TooltipModule } from 'primeng/tooltip'
import { MenubarModule } from 'primeng/menubar'
import { SplitButtonModule } from 'primeng/splitbutton'
import { ProgressSpinnerModule } from 'primeng/progressspinner'
import { ToastModule } from 'primeng/toast'

describe('AppComponent', () => {
    beforeEach(async(() => {
        TestBed.configureTestingModule({
            imports: [RouterTestingModule, TooltipModule, MenubarModule, SplitButtonModule, ProgressSpinnerModule, ToastModule],
            declarations: [AppComponent],
        }).compileComponents()
    }))

    it('should create the app', () => {
        const fixture = TestBed.createComponent(AppComponent)
        const app = fixture.debugElement.componentInstance
        expect(app).toBeTruthy()
    })

    it(`should have as title 'stork'`, () => {
        const fixture = TestBed.createComponent(AppComponent)
        const app = fixture.debugElement.componentInstance
        expect(app.title).toEqual('stork')
    })

    it('should render title', () => {
        const fixture = TestBed.createComponent(AppComponent)
        fixture.detectChanges()
        const compiled = fixture.debugElement.nativeElement
        expect(compiled.querySelector('.content span').textContent).toContain('stork app is running!')
    })
})
