package entitymigrator

// Represents an object that runs a continuous process and its execution can be
// paused and unpaused. It is implemented by the PeriodicExecutor struct that
// is embedded in all pullers.
type Pauser interface {
	Pause()
	Unpause()
}
