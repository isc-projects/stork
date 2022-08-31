import { ComponentFixture, TestBed } from '@angular/core/testing'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { ButtonModule } from 'primeng/button'
import { ChipModule } from 'primeng/chip'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TableModule } from 'primeng/table'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { ConfigCheckerPreferencePickerComponent } from './config-checker-preference-picker.component'

describe('ConfigCheckerPreferencePickerComponent', () => {
    let component: ConfigCheckerPreferencePickerComponent
    let fixture: ComponentFixture<ConfigCheckerPreferencePickerComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [TableModule, ChipModule, OverlayPanelModule, NoopAnimationsModule, ButtonModule],
            declarations: [HelpTipComponent, ConfigCheckerPreferencePickerComponent],
        }).compileComponents()
    })

    beforeEach(() => {
        fixture = TestBed.createComponent(ConfigCheckerPreferencePickerComponent)
        component = fixture.componentInstance
        fixture.detectChanges()
    })

    it('should create', () => {
        expect(component).toBeTruthy()
    })

    it('should display a spinner on loading state', () => {
        component.loading = true
        fixture.detectChanges()

        // Displays a spinner icon.
        const icon = fixture.debugElement.query(By.css('.pi-spinner'))
        expect(icon).not.toBeNull()
        expect(icon.classes["pi-spin"]).toBeDefined()
    })

    it('should display a message on empty state', async () => {
        component.checkers = []
        component.loading = false
        fixture.detectChanges()
        const nativeElement = fixture.nativeElement as HTMLElement
        expect(nativeElement.innerText).toContain("There are no checkers.")
    })

    it('should display full layout by default', () => {
        expect(component.minimal).toBeFalse()
    })

    it('should display only necessary columns in a minimal layout', () => {
        component.minimal = true
        component.checkers = [
            {
                name: "n1",
                selectors: ["s1", "s2"],
                globalEnabled: true,
                state: 'disabled',
                triggers: ["t1", "t2", "t3"]
            },
            {
                name: "n2",
                selectors: ["s3"],
                globalEnabled: false,
                state: 'enabled',
                triggers: ["t4"]
            }
        ]

        fixture.detectChanges()
        
        const hiddenClasses = [
            ".picker__description-column",
            ".picker__selector-column",
            ".picker__trigger-column"
        ]

        for (const hiddenClass of hiddenClasses) {
            const elements = fixture.debugElement.queryAll(By.css(hiddenClass))
            // Header and content
            expect(elements.length).toBe(3)
            for (const element of elements) {
                expect(element.nativeElement.clientWidth).toBe(0)
                expect(element.nativeElement.clientHeight).toBe(0)
            }
        }

        // Find all headers and cells.
        const headers = fixture.debugElement.queryAll(By.css("th"))
        const cells = fixture.debugElement.queryAll(By.css("td"))
        const elements = [...headers, ...cells]
            
        // Filter out elements containing the hidden classes.
        const candidates = []
        for (const element of elements) {
            let isHidden = false
            for (const [elementClass, enabled] of Object.entries(element.classes)) {
                if (!enabled) {
                    continue
                }
                if (hiddenClasses.includes('.' + elementClass)) {
                    isHidden = true
                    break
                }
            }
            if (!isHidden) {
                candidates.push(element)
            }
        }

        // Two visible columns for header and content rows.
        expect(candidates.length).toBe(2*(1+2))

        // Check if headers are visible
        for (const element of candidates) {
            expect(element.nativeElement.clientWidth).not.toBe(0)
            expect(element.nativeElement.clientHeight).not.toBe(0)
        }

    })

    it('should correctly cycle the checker state', () => {
        fail("not implemented")
    })

    it('should display the checker description', () => {
        fail("not implemented")
    })

    it('should display the checker selectors', () => {
        fail("not implemented")
    })

    it('should display the checker triggers', () => {
        fail("not implemented")
    })

    it('should activate the submit button only if any changes were provided', () => {
        fail("not implemented")
    })

    it('should detect reverting changes', () => {
        fail("not implemented")
    })

    it('should display the checker state using a color and a proper checkbox state', () => {
        fail("not implemented")
    })

    it('should display inherit state with a global enabled status', () => {
        fail("not implemented")
    })

    it('should handle submitting and set the loading state', () => {
        fail("not implemented")
    })

    it('should handle the reset button', () => {
        fail("not implemented")
    })

    it('should present the help button', () => {
        fail("not implemented")
    })
})