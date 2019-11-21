import { Injectable } from '@angular/core';
import { BehaviorSubject } from 'rxjs';

@Injectable({
  providedIn: 'root'
})
export class LoadingService {

    counter = 0
    texts = []

    constructor() { }

    private loadInProgress = new BehaviorSubject({state: false, text: ''});

    start(text) {
        this.texts.push(text)
        this.counter += 1
        this.loadInProgress.next({
            state: true,
            text: this.texts.join('\n'),
        });
    }

    stop(text) {
        for(let i = 0; i < this.texts.length; i++){
            if (this.texts[i] === text) {
                this.texts.splice(i, 1)
                break
            }
        }
        this.counter -= 1
        if (this.counter < 0) {
            console.info('negative counter', this.counter)
            this.counter = 0
        }

        if (this.counter === 0) {
            this.loadInProgress.next({
                state: false,
                text: '',
            });
        }
    }

    getState() {
        return this.loadInProgress.asObservable();
    }
}
