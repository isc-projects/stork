package storkutil

import (
	"sync"
)

// PausablePoolPausedError is an error that is returned when new task
// is submitted but the pool is paused.
type PausablePoolPausedError struct{}

// Returns the error message.
func (e *PausablePoolPausedError) Error() string {
	return "pausable pool is paused"
}

// PausablePoolStoppedError is an error that is returned when new task
// is submitted but the pool is stopped.
type PausablePoolStoppedError struct{}

// Returns the error message.
func (e *PausablePoolStoppedError) Error() string {
	return "pausable pool is stopped"
}

// Type of the signal sent to the workers to pause or resume.
type pausablePoolCtrlSignal int

const (
	pausablePoolCtrlSignalPause pausablePoolCtrlSignal = iota
	pausablePoolCtrlSignalResume
)

// PausablePool is a pool of workers with the capability to pause and resume.
type PausablePool struct {
	// Worker receives the tasks on this channel.
	tasks chan func()
	// Worker receives the pause signals over these channels. There is one
	// channel per worker.
	ctrlSignals []chan pausablePoolCtrlSignal
	// Indicates that the pool is paused.
	paused bool
	// Indicates that the pool is stopped.
	stopped bool
	// Mutex to protect against concurrent calls to control functions.
	mutex sync.Mutex
}

// Instantiates a new pool with the specified number of workers.
func NewPausablePool(size int) *PausablePool {
	pool := &PausablePool{
		tasks:       make(chan func()),
		ctrlSignals: make([]chan pausablePoolCtrlSignal, size),
		paused:      false,
		stopped:     false,
		mutex:       sync.Mutex{},
	}
	// Initialize the wait group to be waited for starting the pool.
	var wg sync.WaitGroup
	wg.Add(size)
	for i := 0; i < size; i++ {
		pool.ctrlSignals[i] = make(chan pausablePoolCtrlSignal)
		go pool.worker(&wg, i)
	}
	// Ensure that all workers are started before returning.
	wg.Wait()
	return pool
}

// Worker function reading the tasks from the task channel and executing them.
// It also receives the pause signal over the channels. The wg parameter is used
// to wait for all the workers to start before returning the PausablePool instance.
// The i parameter is the index of the worker. It is used to select the correct
// control signal channel.
func (p *PausablePool) worker(wg *sync.WaitGroup, i int) {
	wg.Done()
	for {
		select {
		case signal, ok := <-p.ctrlSignals[i]:
			if !ok {
				// The channel is closed when the pool is stopped.
				return
			}
			// Ignore unpause signals, as we're not paused here.
			if signal != pausablePoolCtrlSignalPause {
				continue
			}
			// Wait for the resume signal in the inner loop.
			for {
				signal, ok := <-p.ctrlSignals[i]
				if !ok {
					// The channel is closed when the pool is stopped.
					return
				}
				if signal == pausablePoolCtrlSignalResume {
					// Expect the resume signal in the inner loop.
					break
				}
			}
		case task, ok := <-p.tasks:
			if !ok {
				// The channel is closed when the pool is stopped.
				return
			}
			// The worker is not paused. Let's execute the next task.
			task()
		}
	}
}

// Pause the pool and wait for all workers to pause.
func (p *PausablePool) Pause() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.stopped {
		return &PausablePoolStoppedError{}
	}
	if !p.paused {
		p.paused = true
		// Send the pause signal to all workers and wait for them to pause.
		for _, c := range p.ctrlSignals {
			c <- pausablePoolCtrlSignalPause
		}
	}
	return nil
}

// Resume the pool with protecting from concurrent access.
func (p *PausablePool) Resume() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.stopped {
		return &PausablePoolStoppedError{}
	}
	if p.paused {
		p.paused = false
		// Send the resume signal to all workers.
		for _, c := range p.ctrlSignals {
			c <- pausablePoolCtrlSignalResume
		}
	}
	return nil
}

// Stop the pool and close all the channels.
func (p *PausablePool) Stop() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if !p.stopped {
		p.stopped = true
		for _, c := range p.ctrlSignals {
			close(c)
		}
		close(p.tasks)
	}
}

// Submits a new task to the pool.
func (p *PausablePool) Submit(task func()) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	switch {
	case p.paused:
		return &PausablePoolPausedError{}
	case p.stopped:
		return &PausablePoolStoppedError{}
	default:
		p.tasks <- task
		return nil
	}
}
