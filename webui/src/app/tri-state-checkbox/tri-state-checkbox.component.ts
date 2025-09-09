import { Component, input, model } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { Checkbox } from 'primeng/checkbox'

@Component({
    selector: 'app-tri-state-checkbox',
    standalone: true,
    imports: [FormsModule, Checkbox],
    templateUrl: './tri-state-checkbox.component.html',
    styleUrl: './tri-state-checkbox.component.sass',
})
export class TriStateCheckboxComponent {
    value = model<boolean | null>(null)
    inputID = input<string | undefined>(undefined)
    disabled = input<boolean>(false)
    label = input<string | undefined>(undefined)

    toggleValues() {
        if (this.value() === true) {
            this.value.set(false)
        } else if (this.value() === false) {
            this.value.set(null)
        } else {
            this.value.set(true)
        }
    }

    onClick() {
        this.toggleValues()
    }

    /**
     *
     * @param event
     */
    onKeyDown(event: KeyboardEvent) {
        if (event.key === 'Enter' || event.key === 'Space') {
            this.toggleValues()
            event.preventDefault()
        }
    }
}
