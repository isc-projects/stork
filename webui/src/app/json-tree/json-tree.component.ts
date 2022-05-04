import { KeyValue } from '@angular/common'
import { Component, Input, TemplateRef } from '@angular/core'

/**
 * Typing for page changed event of PrimeNG navigation.
 * PrimeNG doesn't contain specific type, but returns any.
 */
interface PageChangedEvent {
    /** Index of first record */
    first: number
    /** Number of rows to display in new page */
    rows: number
    /** Index of the new page */
    page: number
    /** Total number of pages */
    pageCount: number
}

/**
 * Component providing a general-purpose JSON viewer.
 *
 * The component presents data in tree form with nested, collapsible nodes.
 * It accepts any Javascript object, but was designed to work with plain
 * objects with JSON-subset member types.
 *
 * It supports vertically and horizontally large objects using pagination and lazy-loading.
 */
@Component({
    selector: 'app-json-tree',
    templateUrl: './json-tree.component.html',
    styleUrls: ['./json-tree.component.sass'],
})
export class JsonTreeComponent {
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

        // Count of children (subnodes)
        if (this.isComplex()) {
            this.totalChildrenCount = Object.keys(value).length
        }

        // Set children comparator for proper ordering.
        if (this.isArray()) {
            this._keyValueComparator = JsonTreeComponent._arrayKeyValueComparator
        } else if (this.isObject()) {
            this._keyValueComparator = JsonTreeComponent._objectKeyValueComparator
        }
    }

    /**
     * Return value to display
     */
    get value() {
        return this._value
    }

    private _key: string = null

    /**
     * Current value key set via recursion.
     *
     * If it is null or undefined it is a top level component. In this case, the key elements are omitted.
     * This function must only be called internally via recursion.
     */
    @Input()
    set key(key) {
        this._key = key
    }

    /** Return current value's key */
    get key() {
        return this._key
    }

    private _autoExpandMaxNodeCount = 0

    /**
     * Complex nodes (as node of object or array) will be initially opened if number of
     * sub-keys is less or equal @_autoExpandMaxNodeCount value.
     *
     * 0 for disable. Number.MAX_SAFE_INTEGER for initially open all nodes.
     *
     * Note: if node contains exactly one subnode then it is always opened.
     *
     * @param maxNodeCount Maximum number nodes that can be automatically expanded.
     *                     Set to 0 to disable auto expanding. Set to
     *                     Number.MAX_SAFE_INTEGER to initially open all nodes.
     */
    @Input()
    set autoExpandMaxNodeCount(maxNodeCount: number) {
        this._autoExpandMaxNodeCount = maxNodeCount
    }

    /**
     * Maximal number of subnodes when current level should be initially opened.
     * This value is pass through levels below.
     */
    get autoExpandMaxNodeCount() {
        return this._autoExpandMaxNodeCount
    }

    private _forceOpenThisLevel = false

    /**
     * Force open node of this component.
     */
    @Input()
    set forceOpenThisLevel(opened: boolean) {
        this._forceOpenThisLevel = opened
    }

    /**
     * If true then the current node should be always opened.
     */
    get forceOpenThisLevel(): boolean {
        return this._forceOpenThisLevel
    }

    private _recursionLevel = 0

    /**
     * Counter of recursion level (0 is top). It is used to handle very nested
     * objects or cyclic objects.
     *
     * When recursion level is equal to @_maxRecursionLevel then nested components
     * aren't rendered. User can force rendering next nested nodes.
     * In this case the recursion level is reset.
     */
    @Input()
    set recursionLevel(level: number) {
        this._recursionLevel = level
    }

    /**
     * Current recursion level.
     * This value is decreased and pass through levels below.
     */
    get recursionLevel() {
        return this._recursionLevel
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
     * Enable/disable showing a secret value after a click on the placeholder
     */
    @Input()
    set canShowSecrets(enabled: boolean) {
        this._canShowSecrets = enabled
    }

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

    /** First child index to display using pagination. */
    get childStart() {
        return this._childStart
    }

    /**
     * Specifies end child index for pagination.
     * Child with this index isn't displayed.
     */
    get childEnd() {
        return this._childEnd
    }

    /** Maximum number of children on a paginated page. */
    get childStep() {
        return this._childStep
    }

    /** Key-value comparator for order children nodes. */
    get keyValueComparator() {
        return this._keyValueComparator
    }

    /**
     * Maximal number of recursions - nesting levels to render.
     * This value protects from infinite render cyclic object (which causes crash
     * or freeze the browser) or decrease performance on very nested objects.
     */
    private _maxRecursionLevel = 50

    /**
     * Number of sub-nodes to render.
     * Valid only for complex type of @_value (object or array).
     */
    private _childStart = 0

    /**
     * Maximal number of children (sub-nodes) to render.
     * If @_value has more subnodes then paginator is shown.
     * Valid only for complex type of @_value (object or array).
     *
     * The default value of 50 seems sufficient to show most of the nodes
     * in Stork without pagination.
     */
    private _childStep = 50

    /**
     * Number of end child (sub-node) to render.
     * Valid only for complex type of @_value (object or array).
     */
    private _childEnd = this._childStep

    /**
     * Indicate when next page of pagination is loaded after
     * click on paginator button or input page number in jump-to-page control.
     * Valid only for complex type of @_value (object or array) which
     * number of children exceeds @_childStep.
     *
     * This variable isn't used when component is loaded for the first time.
     */
    private _childrenLoading = false

    /**
     * Number of children (subnodes).
     * Valid only for complex type of @_value (object or array). Otherwise 0.
     *
     * This variable is accessible directly (without property) due recommendation
     * of PrimeNG pagination authors.
     */
    totalChildrenCount = 0

    /**
     * It specifies how subnodes are ordered.
     * Valid only for complex type of @_value (object or array). For other
     * types it is null.
     *
     * Arrays are ordered by index.
     * Rest of objects are ordered by key.
     */
    private _keyValueComparator?: (a: KeyValue<number | string, any>, b: KeyValue<number | string, any>) => number =
        null
    /** Order key-value of array by index */
    private static _arrayKeyValueComparator = (a: KeyValue<number, any>, b: KeyValue<number, any>) => a.key - b.key
    /** Order key-value of object by key */
    private static _objectKeyValueComparator = (a: KeyValue<string, any>, b: KeyValue<string, any>) =>
        a.key.localeCompare(b.key)

    /**
     * Handle reset recursion button click event.
     * It causes that next @_maxRecursionLevel nested
     * nodes will be rendered.
     */
    onClickResetRecursionLevel() {
        this._recursionLevel = 0
    }

    /**
     * Handle change page event of PrimeNG paginator.
     * It is used for paginate subnodes.
     * @param ev PageChangedEvent
     */
    onPageChildrenChanged(ev: PageChangedEvent) {
        this.onEnterJumpToPage(ev.page)
    }

    /**
     * Handle change page of number input (extra addon to PrimeNG paginator).
     * @page parameter may be raw user input then it needs to be parse.
     * This handler sets loading state of subnode elements and calculate
     * child limits to display.
     *
     * @param page number Requested page to display
     */
    onEnterJumpToPage(page: number) {
        // "Rows" is word used by PrimeNG.
        const rows = this._childStep
        const childCount = this.totalChildrenCount

        // Page may come from a user input. Ensure it is not a floating point value.
        page = Math.round(page)

        // Check integer overflow. User may provide any number.
        const maxPage = Math.max(0, Math.ceil(childCount / rows) - 1)

        if (page > maxPage) {
            page = maxPage
        } else if (page < 0) {
            page = 0
        }

        // Calculate child limits
        let childStart = page * rows
        let childEnd = childStart + rows

        // Check if child limits exceed maximal or minimal number of children.
        if (childStart < 0) {
            childStart = 0
            childEnd = childStart + rows
        }
        if (childEnd > childCount) {
            childEnd = childCount
        }

        // Set loading state
        this._childrenLoading = true

        // It must be delayed, because first should be set loading state
        // and next new subnodes should be render.
        // If both are in the same Angular "change" then loading state is loose.
        setTimeout(() => {
            this._childStart = childStart
            this._childEnd = childEnd
        }, 0)
    }

    /**
     * Handle rendering finish event.
     *
     * It is only used when a user changes the page using pagination. It isn't used
     * when children are rendered for the first time.
     */
    onFinishRenderChildren() {
        // This event is valid only when loading state is set.
        if (this._childrenLoading) {
            // This event isn't handle exactly when rendering is finished
            // but when Angular for-loop of children components is finished.
            // It delay change of loading state. It will be done when current
            // Angular render-loop will be finished.
            setTimeout(() => {
                this._childrenLoading = false
            })
        }
    }

    /**
     * Specifies when value to display @_value is date
     */
    isDate(): boolean {
        return this._value instanceof Date
    }

    /**
     * Specifies when a value to display @_value has primitive type.
     *
     * Primitive is each type which isn't object (included array).
     * Primitives are: number, string, boolean, null, undefined,
     * but also date (object of Date).
     *
     * @returns boolean
     */
    isPrimitive(): boolean {
        if (this.isDate()) {
            return true
        }
        return this._value !== Object(this._value)
    }

    /**
     * Specifies when value to display @_value has complex type.
     *
     * Complex is any type which isn't primitive.
     * Complex are plain object, array, instance of class (excluded Date),
     *            function.
     * @returns boolean
     */
    isComplex(): boolean {
        return !this.isPrimitive()
    }

    /**
     * Specifies when value to display @_value is array.
     * It implies that isComplex returns true.
     * @returns boolean
     */
    isArray(): boolean {
        return this._value instanceof Array
    }

    /**
     * Specifies when value to display @_value is array.
     * Strings will be display with quotes.
     * @returns boolean
     */
    isString(): boolean {
        return typeof this._value === 'string'
    }

    /**
     * Specifies when value to display @_value is number.
     */
    isNumber(): boolean {
        return typeof this._value === 'number'
    }

    /**
     * Specifies when value to display @_value is empty.
     * Empty may be only complex type. Empty contains no keys
     * (hidden, not enumerable keys aren't included).
     *
     * Empty will be displayed as primitive value, but with
     * specific brackets.
     *
     * Only complex types may be empty (primitive value isn't empty).
     *
     * @returns boolean
     */
    isEmpty(): boolean {
        return this.isComplex() && Object.keys(this._value).length === 0
    }

    /**
     * Specifies when value to display @_value is object.
     * @returns boolean
     */
    isObject(): boolean {
        return this.isComplex() && !this.isArray()
    }

    /**
     * Specifies when value to display @_value is null or undefined.
     */
    isNullOrUndefined(): boolean {
        return this._value == null
    }

    /**
     * Specifies if this component instance represents a root level.
     *
     * The root level component isn't nested in other json-tree-components.
     * and has no key. There must be exactly one root level component instance
     * in the JSON viewer. All sub-nodes must have keys.
     *
     * The root level component can't be collapsed.
     *
     * @returns boolean value indicating if the component represents the root
     *          level.
     */
    isRootLevel(): boolean {
        return !this._key
    }

    /**
     * Specifies when the node represented by this component instance has a value.
     *
     * It is intended for handling non-initialized viewer state,
     * when parent component has not fetched the value yet.
     *
     * @returns true if the node represented by this component instance has been
     *          assigned.
     */
    hasAssignedValue(): boolean {
        return !this.isRootLevel() || !this.isNullOrUndefined()
    }

    /**
     * Specifies when current level should be initially opened.
     *
     * The node state is managed internally by HTML, but initial state may
     * be opened or closed.
     *
     * Node is initially opened if it is forced by parent component
     * (when current node has no siblings) or the number of children is
     * less or equal @_autoExpandMaxNodeCount.
     *
     * Top level is always initially opened, but it is never indented
     * and cannot be collapsed.
     *
     * @returns true if the node is initially opened.
     */
    isInitiallyOpened(): boolean {
        return this._forceOpenThisLevel || this.totalChildrenCount <= this._autoExpandMaxNodeCount
    }

    /**
     * Specifies when the value to display is corrupted.
     *
     * When a node is corrupted its key is marked with a red wave.
     *
     * This is an experimental function and may result in false alarms.
     * It is intended to help find places when a value was manually
     * edited and some characters (like quotes) were lost.
     *
     * Node is corrupted when:
     * For string:
     *     - A number of open and close curly or square brackets aren't the same.
     *     - A number of single or double quotes is odd.
     * For arrays:
     *     - Items have different types.
     *
     * @returns true if the node value is considered corrupted.
     */
    isCorrupted(): boolean {
        if (typeof this._value === 'string') {
            const parityTestBrackets = [
                ['{', '}'],
                ['[', ']'],
            ]
            const parityTestItems = ['"', `'`]
            const characters = [...parityTestItems, ...([] as string[]).concat(...parityTestBrackets)]
            const countDict = {}

            // Count specific characters
            for (const c of this._value) {
                if (characters.indexOf(c) !== -1) {
                    if (!countDict.hasOwnProperty(c)) {
                        countDict[c] = 0
                    }
                    countDict[c] += 1
                }
            }

            // Check if any bracket has no pair
            return (
                parityTestBrackets.some(
                    ([left, right]) => countDict[left] !== countDict[right]
                    // check if each item isn't pair
                ) || parityTestItems.some((item) => countDict[item] % 2 === 1)
            )
        } else if (this._value instanceof Array && this._value.length > 0) {
            const referenceType = typeof this._value[0]
            return this._value.some((v) => typeof v !== referenceType)
        }
        return false
    }

    /**
     * Specifies when recursion level is reached.
     *
     * It is used to avoid performance problems on cyclic objects.
     *
     * @returns true if recursion level was reached.
     */
    isRecursionLevelReached(): boolean {
        return this._recursionLevel >= this._maxRecursionLevel
    }

    /**
     * Specifies when the value is a secret and should be hidden using a placeholder.
     */
    isSecret(): boolean {
        return this.secretKeys.includes(this.key)
    }

    /**
     * Specifies when the value of leaf should be rendered using custom template
     *
     * @returns true if the value should be rendered using custom template
     */
    hasCustomValueTemplate(): boolean {
        return this.key in this.customValueTemplates
    }

    /**
     * Specifies when value to display has only one child.
     *
     * Only complex type value can have a single child.
     *
     * @returns true if the node has a single child.
     */
    hasSingleChild(): boolean {
        return this.totalChildrenCount === 1
    }

    /**
     * Specifies when the node has too many children to display at once.
     *
     * @returns true if the value has to be paginated.
     */
    hasPaginateChildren(): boolean {
        return this.totalChildrenCount > this._childStep
    }

    /**
     * Specifies if the paginated children of the component are loading.
     *
     * @returns true if the paginated children are loading.
     */
    areChildrenLoading() {
        return this._childrenLoading
    }
}
