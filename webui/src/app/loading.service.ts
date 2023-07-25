import { Injectable } from '@angular/core'
import { BehaviorSubject } from 'rxjs'

/**
 * Global loading service.
 *
 * Note: It seems to be unused from the beginning. Use with caution.
 */
@Injectable({
    providedIn: 'root',
})
export class LoadingService {
    counter = 0
    texts = []

    private loadInProgress = new BehaviorSubject({ state: false, text: '' })

    /**
     * Requests to start the global loading.
     * @param text Text to display
     */
    start(text) {
        this.texts.push(text)
        this.counter += 1
        this.loadInProgress.next({
            state: true,
            text: this.texts.join('\n'),
        })
    }

    /**
     * Requests to stop the global loading. All callers of the @start method
     * must call the @stop method to stop loading.
     * @param text Text to display
     */
    stop(text) {
        for (let i = 0; i < this.texts.length; i++) {
            if (this.texts[i] === text) {
                this.texts.splice(i, 1)
                break
            }
        }
        this.counter -= 1
        if (this.counter < 0) {
            this.counter = 0
        }

        if (this.counter === 0) {
            this.loadInProgress.next({
                state: false,
                text: '',
            })
        }
    }

    /** Returns the loading state as an observable. */
    getState() {
        return this.loadInProgress.asObservable()
    }
}
