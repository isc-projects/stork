import { ComponentFixture, TestBed } from '@angular/core/testing'
import { By } from '@angular/platform-browser'
import { NoopAnimationsModule } from '@angular/platform-browser/animations'
import { ButtonModule } from 'primeng/button'
import { ChipModule } from 'primeng/chip'
import { OverlayPanelModule } from 'primeng/overlaypanel'
import { TableModule } from 'primeng/table'
import { ConfigChecker } from '../backend'
import { HelpTipComponent } from '../help-tip/help-tip.component'
import { ConfigCheckerPreferencePickerComponent } from './config-checker-preference-picker.component'
import { TriStateCheckboxModule } from 'primeng/tristatecheckbox'
import { FormsModule } from '@angular/forms'
import { TagModule } from 'primeng/tag'

describe('ConfigCheckerPreferencePickerComponent', () => {
    let component: ConfigCheckerPreferencePickerComponent
    let fixture: ComponentFixture<ConfigCheckerPreferencePickerComponent>

    beforeEach(async () => {
        await TestBed.configureTestingModule({
            imports: [
                TableModule,
                ChipModule,
                OverlayPanelModule,
                NoopAnimationsModule,
                ButtonModule,
                FormsModule,
                TriStateCheckboxModule,
                TagModule,
            ],
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
        const icon = fixture.debugElement.query(By.css('.p-icon-spin'))
        expect(icon).not.toBeNull()
    })

    it('should display a message on empty state', async () => {
        component.checkers = []
        component.loading = false
        fixture.detectChanges()
        const nativeElement = fixture.nativeElement as HTMLElement
        expect(nativeElement.innerText).toContain('There are no checkers enabled.')
    })

    it('should display full layout by default', () => {
        expect(component.minimal).toBeFalse()
    })

    it('should display only necessary columns in a minimal layout', () => {
        component.minimal = true
        component.checkers = [
            {
                name: 'n1',
                selectors: ['s1', 's2'],
                globallyEnabled: true,
                state: 'disabled',
                triggers: ['t1', 't2', 't3'],
            },
            {
                name: 'n2',
                selectors: ['s3'],
                globallyEnabled: false,
                state: 'enabled',
                triggers: ['t4'],
            },
        ]

        fixture.detectChanges()

        const hiddenClasses = ['.picker__description-column', '.picker__selector-column', '.picker__trigger-column']

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
        const headers = fixture.debugElement.queryAll(By.css('th'))
        const cells = fixture.debugElement.queryAll(By.css('td'))
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
        expect(candidates.length).toBe(2 * (1 + 2))

        // Check if the elements are visible
        for (const element of candidates) {
            expect(element.nativeElement.clientWidth).not.toBe(0)
            expect(element.nativeElement.clientHeight).not.toBe(0)
        }
    })

    it('should correctly cycle the checker state', () => {
        const checker: ConfigChecker = {
            name: 'foo',
            state: 'disabled',
            globallyEnabled: true,
            selectors: [],
            triggers: [],
        }

        component.checkers = [checker]
        component.allowInheritState = true

        component.onCheckerStateChanged(checker)
        expect(component.getActualState(checker)).toBe('inherit')
        component.onCheckerStateChanged(checker)
        expect(component.getActualState(checker)).toBe('enabled')
        component.onCheckerStateChanged(checker)
        expect(component.getActualState(checker)).toBe('disabled')

        component.allowInheritState = false

        component.onCheckerStateChanged(checker)
        expect(component.getActualState(checker)).toBe('enabled')
        component.onCheckerStateChanged(checker)
        expect(component.getActualState(checker)).toBe('disabled')
    })

    it('should display the checker description', () => {
        component.checkers = [
            {
                globallyEnabled: true,
                name: 'host_cmds_presence',
                selectors: [],
                state: 'enabled',
                triggers: [],
            },
        ]
        fixture.detectChanges()

        const element = fixture.debugElement.query(By.css('td.picker__description-column'))
        expect(element).not.toBeNull()
        const content = (element.nativeElement as HTMLElement).innerText
        expect(content).toContain(
            'This checker verifies whether the host_cmds hook library is loaded when host backend is in use.'
        )
    })

    it('should display the checker selectors', () => {
        component.checkers = [
            {
                globallyEnabled: true,
                name: 'foo',
                selectors: ['each-daemon', 'foobar'],
                state: 'enabled',
                triggers: [],
            },
        ]
        fixture.detectChanges()

        const element = fixture.debugElement.query(By.css('td.picker__selector-column'))
        expect(element).not.toBeNull()
        const content = (element.nativeElement as HTMLElement).innerText
        expect(content).toContain('each-daemon')
        expect(content).toContain('foobar')
    })

    it('should display the checker triggers', () => {
        component.checkers = [
            {
                globallyEnabled: true,
                name: 'foo',
                selectors: [],
                state: 'enabled',
                triggers: ['host reservations change', 'barfoo'],
            },
        ]
        fixture.detectChanges()

        const element = fixture.debugElement.query(By.css('td.picker__trigger-column'))
        expect(element).not.toBeNull()
        const content = (element.nativeElement as HTMLElement).innerText
        expect(content).toContain('host reservations change')
        expect(content).toContain('barfoo')
    })

    it('should activate the submit button only if any changes were provided', () => {
        component.checkers = [
            {
                globallyEnabled: true,
                name: 'foo',
                selectors: [],
                state: 'enabled',
                triggers: [],
            },
        ]
        component.allowInheritState = true
        fixture.detectChanges()

        const checker = component.checkers[0]
        let submitButton = fixture.debugElement.query(By.css('button[label=Submit]'))
        expect(submitButton).not.toBeNull()

        // No changes
        expect(component.hasChanges).toBeFalse()
        expect(submitButton.attributes).toEqual(
            jasmine.objectContaining({
                disabled: '',
            })
        )

        // Significant changes.
        // Disabled state.
        component.onCheckerStateChanged(checker)
        fixture.detectChanges()
        expect(component.hasChanges).toBeTrue()
        submitButton = fixture.debugElement.query(By.css('button[label=Submit]'))
        expect(submitButton.attributes).not.toEqual(
            jasmine.objectContaining({
                disabled: '',
            })
        )
        // Inherit state.
        component.onCheckerStateChanged(checker)
        fixture.detectChanges()
        expect(component.hasChanges).toBeTrue()
        submitButton = fixture.debugElement.query(By.css('button[label=Submit]'))
        expect(submitButton.attributes).not.toEqual(
            jasmine.objectContaining({
                disabled: '',
            })
        )

        // Revert changes.
        component.onCheckerStateChanged(checker)
        fixture.detectChanges()
        expect(component.hasChanges).toBeFalse()
        submitButton = fixture.debugElement.query(By.css('button[label=Submit]'))
        expect(submitButton.attributes).toEqual(
            jasmine.objectContaining({
                disabled: '',
            })
        )
    })

    it('should the checker state cell should have a proper CSS class', () => {
        const checker = {
            globallyEnabled: true,
            name: 'foo',
            selectors: [],
            state: ConfigChecker.StateEnum.Enabled,
            triggers: [],
        }
        component.checkers = [checker]
        fixture.detectChanges()

        const stateCell = fixture.debugElement.query(By.css('.picker__state-cell'))
        expect(stateCell).not.toBeNull()

        // Enabled state.
        expect(stateCell.classes['picker__state-cell--enabled']).toBeTrue()
        expect(stateCell.classes['picker__state-cell--disabled']).not.toBeTrue()
        expect(stateCell.classes['picker__state-cell--inherit-enabled']).not.toBeTrue()
        expect(stateCell.classes['picker__state-cell--inherit-disabled']).not.toBeTrue()

        // Disabled state.
        checker.state = 'disabled'
        fixture.detectChanges()
        expect(stateCell.classes['picker__state-cell--enabled']).not.toBeTrue()
        expect(stateCell.classes['picker__state-cell--disabled']).toBeTrue()
        expect(stateCell.classes['picker__state-cell--inherit-enabled']).not.toBeTrue()
        expect(stateCell.classes['picker__state-cell--inherit-disabled']).not.toBeTrue()

        // Inherit state.
        // Globally enabled.
        checker.state = 'inherit'
        fixture.detectChanges()
        expect(stateCell.classes['picker__state-cell--enabled']).not.toBeTrue()
        expect(stateCell.classes['picker__state-cell--disabled']).not.toBeTrue()
        expect(stateCell.classes['picker__state-cell--inherit-enabled']).toBeTrue()
        expect(stateCell.classes['picker__state-cell--inherit-disabled']).not.toBeTrue()
        // Globally disabled.
        checker.globallyEnabled = false
        fixture.detectChanges()
        expect(stateCell.classes['picker__state-cell--enabled']).not.toBeTrue()
        expect(stateCell.classes['picker__state-cell--disabled']).not.toBeTrue()
        expect(stateCell.classes['picker__state-cell--inherit-enabled']).not.toBeTrue()
        expect(stateCell.classes['picker__state-cell--inherit-disabled']).toBeTrue()
    })

    it('should display inherit state with a globally enabled status', () => {
        const checker = {
            globallyEnabled: true,
            name: 'foo',
            selectors: [],
            state: ConfigChecker.StateEnum.Inherit,
            triggers: [],
        }
        component.checkers = [checker]
        fixture.detectChanges()

        const stateCell = fixture.debugElement.query(By.css('.picker__state-cell'))
        expect(stateCell).not.toBeNull()

        let content = (stateCell.nativeElement as HTMLElement).textContent
        expect(content.trim()).toEqual('globally enabled')

        checker.globallyEnabled = false
        fixture.detectChanges()
        content = (stateCell.nativeElement as HTMLElement).textContent
        expect(content.trim()).toEqual('globally disabled')
    })

    it('should handle submitting and set the loading state', () => {
        spyOn(component.changePreferences, 'emit')

        component.checkers = [
            {
                globallyEnabled: true,
                name: 'foo',
                selectors: [],
                state: ConfigChecker.StateEnum.Inherit,
                triggers: [],
            },
        ]

        component.onCheckerStateChanged(component.checkers[0])
        component.onSubmit()

        expect(component.loading).toBeTrue()
        expect(component.changePreferences.emit).toHaveBeenCalledOnceWith([
            {
                name: 'foo',
                state: 'enabled',
            },
        ])
    })

    it('should handle the reset button', () => {
        component.checkers = [
            {
                globallyEnabled: true,
                name: 'foo',
                selectors: [],
                state: ConfigChecker.StateEnum.Inherit,
                triggers: [],
            },
        ]

        component.onCheckerStateChanged(component.checkers[0])
        expect(component.getActualState(component.checkers[0])).toBe('enabled')
        component.onReset()
        expect(component.getActualState(component.checkers[0])).toBe('inherit')
    })

    it('should present the help button', () => {
        const helpElement = fixture.debugElement.query(By.directive(HelpTipComponent))
        expect(helpElement).not.toBeNull()
        const helpComponent = helpElement.componentInstance as HelpTipComponent
        expect(helpComponent.subject).toContain('Checkers list')
    })
})
