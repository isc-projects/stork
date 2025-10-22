import { Component, Input } from '@angular/core'
import { CommonModule } from '@angular/common'

/**
 * A component that displays the contents of a text file.
 *
 * It expects that each element of the contents array represents a
 * single file line. The other special characters, like tabs are
 * rendered.
 */
@Component({
    selector: 'app-text-file-viewer',
    standalone: true,
    imports: [CommonModule],
    templateUrl: './text-file-viewer.component.html',
    styleUrl: './text-file-viewer.component.sass',
})
export class TextFileViewerComponent {
    /**
     * Holds the file contents.
     */
    private _contents: string[] = []

    /**
     * Sets the contents with removing the leading and trailing empty lines.
     */
    @Input({ required: true })
    set contents(contents: string[]) {
        // Remove leading empty lines.
        while (contents?.length > 0 && contents[0].trim() === '') {
            contents.shift()
        }
        // Remove trailing empty lines.
        while (contents?.length > 0 && contents[contents.length - 1].trim() === '') {
            contents.pop()
        }
        this._contents = contents ?? []
    }

    /**
     * Returns the file contents as an array of lines.
     */
    get contents(): string[] {
        return this._contents
    }
}
