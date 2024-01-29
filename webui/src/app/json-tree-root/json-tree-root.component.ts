import { Component, Input, OnDestroy, OnInit, TemplateRef } from '@angular/core'
import { Subscription } from 'rxjs'
import { AuthService } from '../auth.service'

/**
 * JSON Tree Component wrapper that minimizes the number of input
 * arguments (by hiding internal ones).
 * It also checks a user group to handle showing/hiding secrets.
 */
@Component({
    selector: 'app-json-tree-root',
    templateUrl: './json-tree-root.component.html',
    styleUrls: [],
})
export class JsonTreeRootComponent implements OnInit, OnDestroy {
    private subscription = new Subscription()

    private _value: any = null

    /**
     * Content of JSON viewer.
     * May be value of any type: primitive (string, number, boolean...), complex (dict, array, object).
     *
     * If it is null or undefined then it means that a proper object is not set yet
     * (for example component initialization in progress). In this case the viewer
     * displays loading indicator (spinner).
     */
    @Input()
    set value(value: any) {
        this._value = value
    }

    /**
     * Return value to display
     */
    get value() {
        return this._value
    }

    private _autoExpand: 'none' | 'all' | number = 'none'

    /**
     * Sets the number of subnodes below which the parent node will be
     * initially opened. Use 'none' value for initially collapse all nodes
     * (except nodes with a single subnode), 'all' for initially expand all
     * nodes, or any positive number to specify an exact number of subnodes
     * to initially open the parent node.
     */
    @Input()
    set autoExpand(state: 'none' | 'all' | number) {
        this._autoExpand = state
    }

    /**
     * Complex nodes (as node of object or array) will be initially opened if number of
     * sub-keys is less or equal a returned value.
     *
     * Note: if node contains exactly one subnode then it is always opened.
     */
    get autoExpandNodeCount(): number {
        if (this._autoExpand === 'none') {
            return 0
        } else if (this._autoExpand === 'all') {
            return Number.MAX_SAFE_INTEGER
        } else if (!isNaN(+this._autoExpand)) {
            return +this._autoExpand
        } else {
            return 50 // Default value
        }
    }

    private _secretKeys = ['password', 'secret']

    /**
     * Set list of secret keys that values will hide using a placeholder.
     * It applies only to a leaf of the tree.
     */
    @Input()
    set secretKeys(keys: string[]) {
        this._secretKeys = keys
    }

    /**
     * Get list of secret keys that values will hide using a placeholder.
     */
    get secretKeys() {
        return this._secretKeys
    }

    private _canShowSecrets = false

    /**
     * Indicates if a secret should be shown after clicking on the placeholder.
     */
    get canShowSecrets() {
        return this._canShowSecrets
    }

    /**
     * Allow render custom content (for example links, labels or glyphs) instead of
     * standard value for specific keys. Custom content is applied only for the leafs.
     */
    @Input()
    customValueTemplates: { [key: string]: TemplateRef<{ key: string; value: string }> } = {}

    constructor(private auth: AuthService) {}

    ngOnInit(): void {
        // Check if user can show the secrets
        this.subscription.add(
            this.auth.currentUser.subscribe(() => {
                this._canShowSecrets = this.auth.superAdmin()
            })
        )
    }

    /**
     * Unsubscribe all subscriptions.
     */
    ngOnDestroy(): void {
        this.subscription.unsubscribe()
    }
}
