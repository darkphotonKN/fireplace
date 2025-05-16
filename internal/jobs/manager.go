package jobs

import (
	"fmt"
	"sync"
)

type Manager struct {
	jobs     []Job
	stopChan chan bool
	mu       sync.Mutex
	wg       sync.WaitGroup
}

type Job interface {
	Start()
	Stop()
}

func NewManager() *Manager {
	return &Manager{
		jobs:     make([]Job, 0),
		stopChan: make(chan bool),
	}
}

// starts all jobs
func (m *Manager) StartAll() {
	for _, job := range m.jobs {
		go func() {
			job.Start()
			<-m.stopChan
			fmt.Println("stopping ongoing channels")
			job.Stop()
			return
		}()
	}

}

// starts all jobs
func (m *Manager) StopAll() {
	close(m.stopChan)
}

func (m *Manager) AddJob(job Job) {
	m.jobs = append(m.jobs, job)
}

func (m *Manager) RemoveJob(job Job) {
	filteredJobs := make([]Job, 0)

	for _, j := range m.jobs {
		if j == job {
			continue
		}
		filteredJobs = append(filteredJobs, job)
	}

	m.jobs = filteredJobs
}
