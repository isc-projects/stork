/**
 * Enumeration for different tab types displayed by a component.
 */
export enum TabType {
    List = 1,
    New,
    Edit,
    Display,
}

/**
 * A class representing the contents of a tab displayed by a component.
 *
 * @tparam FormStateType a type of the form state held in the tab.
 * @tparam TabSubjectType a type of the object holding the data.
 */
export class Tab<FormStateType, TabSubjectType> {
    /**
     * Preserves information specified in a form.
     */
    state: FormStateType

    /**
     * Indicates if the form has been submitted.
     */
    submitted = false

    /**
     * Constructor.
     *
     * @param stateCreator factory function creating default state instance.
     * @param tabType tab type.
     * @param tabSubject information displayed in the tab.
     */
    constructor(
        private stateCreator: { new (): FormStateType },
        public tabType: TabType,
        public tabSubject?: TabSubjectType
    ) {
        this.setTabTypeInternal(tabType)
    }

    /**
     * Sets new tab type and initializes the form accordingly.
     *
     * It is a private function variant that does not check whether the type
     * is already set to the desired value.
     */
    private setTabTypeInternal(tabType: TabType): void {
        switch (tabType) {
            case TabType.New:
            case TabType.Edit:
                this.state = new this.stateCreator()
                break
            default:
                this.state = null
                break
        }
        this.submitted = false
        this.tabType = tabType
    }

    /**
     * Sets new tab type and initializes the form accordingly.
     *
     * It does nothing when the type is already set to the desired value.
     */
    public setTabType(tabType: TabType): void {
        if (this.tabType === tabType) {
            return
        }
        this.setTabTypeInternal(tabType)
    }
}
