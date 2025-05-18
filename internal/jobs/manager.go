package jobs

import (
	"fmt"
	"sync"
)

type Manager struct {
	jobs     []Job
	stopChan chan bool
	mu       sync.Mutex
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
		j := job // prevent variable capture issue
		go func() {
			j.Start()
			<-m.stopChan
			fmt.Println("stopping ongoing channels")
			j.Stop()
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
