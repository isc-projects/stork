import { AnyDaemon } from './backend'

// An interface used by the components tracking opened daemon tabs.
// It embeds the daemon data fetched over the REST API.
export interface DaemonTab {
    daemon: AnyDaemon
}
