import { App } from './backend'

// An interface used by the components tracking opened app tabs.
// It embeds the app data fetched over the REST API.
export interface AppTab {
    app: App
}
