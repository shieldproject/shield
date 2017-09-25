// +build windows

package winsvc

import (
	"errors"
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// Mgr is used to manage Windows services.
type Mgr struct {
	m     *mgr.Mgr
	match func(description string) bool
}

// Connect returns a new Mgr that will monitor all services with descriptions
// matched by match.  If match is nil all services are matched.
func Connect(match func(description string) bool) (*Mgr, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, err
	}
	if match == nil {
		match = func(_ string) bool { return true }
	}
	return &Mgr{m: m, match: match}, nil
}

// Disconnect closes connection to the service control manager m.
func (m *Mgr) Disconnect() error {
	return m.m.Disconnect()
}

func toString(p *uint16) string {
	if p == nil {
		return ""
	}
	return syscall.UTF16ToString((*[4096]uint16)(unsafe.Pointer(p))[:])
}

// serviceDescription, returns the description of service s.
func serviceDescription(s *mgr.Service) (string, error) {
	var p *windows.SERVICE_DESCRIPTION
	n := uint32(1024)
	for {
		b := make([]byte, n)
		p = (*windows.SERVICE_DESCRIPTION)(unsafe.Pointer(&b[0]))
		err := windows.QueryServiceConfig2(s.Handle,
			windows.SERVICE_CONFIG_DESCRIPTION, &b[0], n, &n)
		if err == nil {
			break
		}
		if err.(syscall.Errno) != syscall.ERROR_INSUFFICIENT_BUFFER {
			return "", err
		}
		if n <= uint32(len(b)) {
			return "", err
		}
	}
	return toString(p.Description), nil
}

// services, returns all of the services that match the Mgr's match function.
func (m *Mgr) services() ([]*mgr.Service, error) {
	names, err := m.m.ListServices()
	if err != nil {
		return nil, fmt.Errorf("winsvc: listing services: %s", err)
	}
	var svcs []*mgr.Service
	for _, name := range names {
		s, err := m.m.OpenService(name)
		if err != nil {
			continue // ignore - likely access denied
		}
		desc, err := serviceDescription(s)
		if err != nil {
			s.Close()
			continue // ignore - likely access denied
		}
		if m.match(desc) {
			svcs = append(svcs, s)
		} else {
			s.Close()
		}
	}
	return svcs, nil
}

// iter, calls function fn concurrently on each service matched by Mgr.
// The service is closed for fn and the first error, if any, is returned.
//
// fn must be safe for concurrent use and must not block indefinitely.
func (m *Mgr) iter(fn func(*mgr.Service) error) (first error) {
	svcs, err := m.services()
	if err != nil {
		return err
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(svcs))
	for _, s := range svcs {
		go func(s *mgr.Service) {
			defer wg.Done()
			defer s.Close()
			if err := fn(s); err != nil {
				mu.Lock()
				if first == nil {
					first = err
				}
				mu.Unlock()
			}
		}(s)
	}
	wg.Wait()
	return
}

func svcStartTypeString(startType uint32) string {
	switch startType {
	case mgr.StartManual:
		return "StartManual"
	case mgr.StartAutomatic:
		return "StartAutomatic"
	case mgr.StartDisabled:
		return "StartDisabled"
	}
	return fmt.Sprintf("Invalid Service StartType: %d", startType)
}

func SetStartType(s *mgr.Service, startType uint32) error {
	conf, err := s.Config()
	if err != nil {
		return &ServiceError{"querying config for service", s.Name, err}
	}
	if conf.StartType == startType {
		return nil
	}
	conf.StartType = startType
	if err := s.UpdateConfig(conf); err != nil {
		return &ServiceError{"updating config for service", s.Name, err}
	}
	return nil
}

// querySvc, queries the service status of service s.  This is really here to
// return a formated error message.
func querySvc(s *mgr.Service) (svc.Status, error) {
	status, err := s.Query()
	if err != nil {
		err = &ServiceError{"querying status of service", s.Name, err}
	}
	return status, err
}

// calculateWaitHint, converts a service's WaitHint into a time duration and
// calculates the interval the caller should wait for before rechecking the
// service's status.
//
// If no WaitHint is provided the default of 10 seconds is returned.  As per
// Microsoft's recommendations he returned interval will be between 1 and 10
// seconds.
func calculateWaitHint(status svc.Status) (waitHint, interval time.Duration) {
	//
	// This is all a little confusing, so I included the definition of WaitHint
	// and Microsoft's guidelines on how to use below:
	//
	//
	// Definition of WaitHint:
	//
	//   The estimated time required for a pending start, stop, pause, or
	//   continue operation, in milliseconds. Before the specified amount
	//   of time has elapsed, the service should make its next call to the
	//   SetServiceStatus function with either an incremented dwCheckPoint
	//   value or a change in dwCurrentState. If the amount of time specified
	//   by dwWaitHint passes, and dwCheckPoint has not been incremented or
	//   dwCurrentState has not changed, the service control manager or service
	//   control program can assume that an error has occurred and the service
	//   should be stopped. However, if the service shares a process with other
	//   services, the service control manager cannot terminate the service
	//   application because it would have to terminate the other services
	//   sharing the process as well.
	//
	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms685996(v=vs.85).aspx
	//
	//
	// Using the wait hint to check for state transition:
	//
	//   Do not wait longer than the wait hint. A good interval is
	//   one-tenth of the wait hint but not less than 1 second
	//   and not more than 10 seconds.
	//
	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms686315(v=vs.85).aspx
	//
	waitHint = time.Duration(status.WaitHint) * time.Millisecond
	if waitHint == 0 {
		waitHint = time.Second * 10
	}
	interval = waitHint / 10
	switch {
	case interval < time.Second:
		interval = time.Second
	case interval > time.Second*10:
		interval = time.Second * 10
	}
	return
}

// waitPending, waits for service s to transition out of pendingState, which
// must be either StartPending or StopPending.  A two minute time limit is
// enforced for the state transition.
//
// See calculateWaitHint for an explanation of how the service's WaitHint is
// used to check progress.
func waitPending(s *mgr.Service, pendingState svc.State) (svc.Status, error) {
	// Arbitrary timeout to prevent misbehaving
	// services from triggering an infinite loop.
	const Timeout = time.Minute * 2

	if pendingState != svc.StartPending && pendingState != svc.StopPending {
		// This is a programming error and really should be a panic.
		return svc.Status{}, errors.New("winsvc: invalid pending state: " +
			svcStateString(pendingState))
	}

	status, err := querySvc(s)
	if err != nil {
		return status, err
	}

	start := time.Now()
	checkpoint := start
	oldCheckpoint := status.CheckPoint
	highCPU := 0

	for status.State == pendingState {
		waitHint, interval := calculateWaitHint(status)
		time.Sleep(interval) // sleep before rechecking status

		status, err = querySvc(s)
		if err != nil {
			return status, err
		}
		if status.State != pendingState {
			break
		}

		switch {
		// The service incremented it's checkpoint, reset timer
		case status.CheckPoint > oldCheckpoint:
			checkpoint = time.Now()
			oldCheckpoint = status.CheckPoint

		// No progress made within the wait hint.
		case time.Since(checkpoint) > waitHint:
			// Handle high CPU situations.  This is incredibly crude,
			// but it works!
			switch {
			case cpu.CPU() > 90:
				highCPU = 10
			case highCPU > 0:
				highCPU--
			default:
				err := &TransitionError{
					Msg:      "no progress waiting for state transition",
					Name:     s.Name,
					Status:   status,
					WaitHint: waitHint,
					Duration: time.Since(start),
				}
				return status, err
			}

		// Exceeded our timeout
		case time.Since(start) > Timeout:
			err := &TransitionError{
				Msg:      "timeout waiting for state transition",
				Name:     s.Name,
				Status:   status,
				WaitHint: waitHint,
				Duration: time.Since(start),
			}
			return status, err
		}
	}

	if status.State == pendingState {
		err := &TransitionError{
			Msg:      "failed to transition out of state",
			Name:     s.Name,
			Status:   status,
			Duration: time.Since(start),
		}
		return status, err
	}

	return status, nil
}

func doStart(s *mgr.Service) error {
	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms681383(v=vs.85).aspx
	const ERROR_SERVICE_ALREADY_RUNNING = syscall.Errno(0x420)

	// Set start type to manual to enable starting the service.
	if err := SetStartType(s, mgr.StartManual); err != nil {
		return err
	}

	status, err := querySvc(s)
	if err != nil {
		return err
	}

	// Wait to transition out of any pending states
	if status.State == svc.StopPending || status.State == svc.StartPending {
		status, err = waitPending(s, status.State)
		if err != nil {
			return err
		}
	}

	// Check if the service is already running
	if status.State == svc.Running {
		return nil
	}

	if err := s.Start(); err != nil {
		// Ignore error if the service is running
		if err != ERROR_SERVICE_ALREADY_RUNNING {
			return &ServiceError{"starting service", s.Name, err}
		}
	}

	// Wait for the service to start
	status, err = waitPending(s, svc.StartPending)
	if err != nil {
		return err
	}
	// Failed to start - return a StartError so we know to retry
	if status.State != svc.Running {
		return &StartError{Name: s.Name, Status: status}
	}

	// Make sure we stay running.  I wish we didn't have to do this, but
	// we run our processes with pipe.exe, which means that before WinSW
	// can notice that our process died: pipe.exe must notice it died, do
	// any logging and exit itself.  This is slow and indeterminate.
	//
	for i := 0; i < 5; i++ {
		time.Sleep(time.Millisecond * 100)
		status, err = querySvc(s)
		if err != nil {
			return err
		}
		if status.State != svc.Running {
			return &StartError{Name: s.Name, Status: status}
		}
	}
	return nil
}

func Start(s *mgr.Service) error {
	const Retries = 10
	const Interval = time.Second

	var err error
	for i := 0; i < Retries; i++ {
		err = doStart(s)
		if err == nil {
			break
		}
		if _, ok := err.(*StartError); !ok {
			return err
		}
		time.Sleep(Interval)
	}
	return err
}

// Start starts all of the service monitored by Mgr.
func (m *Mgr) Start() (first error) {
	return m.iter(Start)
}

func Stop(s *mgr.Service) error {
	const Timeout = time.Second * 30

	// Disable service start to prevent flapping services from being
	// automatically restarted while attempting to stop them.
	if err := SetStartType(s, mgr.StartDisabled); err != nil {
		return err
	}

	status, err := querySvc(s)
	if err != nil {
		return err
	}

	// Wait to transition out of any pending states
	if status.State == svc.StopPending || status.State == svc.StartPending {
		status, err = waitPending(s, status.State)
		if err != nil {
			return err
		}
	}

	// Check if the service is already stopped
	if status.State == svc.Stopped {
		return nil
	}

	// Stop service

	status, err = s.Control(svc.Stop)
	if err != nil {
		return &ServiceError{"stopping service", s.Name, err}
	}

	// Check if the returned status is empty
	if status.State == 0 || status.Accepts == 0 {
		status, err = querySvc(s)
		if err != nil {
			return err
		}
	}

	// Wait for service to stop
	if status.State == svc.StopPending {
		status, err = waitPending(s, svc.StopPending)
		if err != nil {
			return err
		}
	}

	// Check that the service is actually stopped
	start := time.Now()
	for status.State != svc.Stopped {
		_, interval := calculateWaitHint(status)
		time.Sleep(interval)

		status, err = querySvc(s)
		if err != nil {
			return err
		}
		if status.State == svc.Stopped {
			break
		}
		if d := time.Since(start); d > Timeout {
			return &ServiceError{"stop service", s.Name, &TimeoutError{Timeout, d}}
		}
	}

	return nil
}

// Stop stops all of the services monitored by Mgr concurrently and waits
// for the stop to complete. Stopping services with dependent services is
// not supported.
func (m *Mgr) Stop() (first error) {
	return m.iter(Stop)
}

func (m *Mgr) doDelete(s *mgr.Service) error {
	const Timeout = time.Second * 60

	st, err := querySvc(s)
	if err != nil {
		return err
	}

	// Stop the service if it is running.  Services that are not stopped
	// can be marked for deletion, but are not actually deleted by the
	// SCM until they stop or the computer restarts.
	if st.State != svc.Stopped {
		if err := Stop(s); err != nil {
			return err
		}
	}

	// Delete the service and immediately close the handle to it
	if err := s.Delete(); err != nil {
		return &ServiceError{"deleting service", s.Name, err}
	}
	name := s.Name
	s.Close() // Close the service otherwise it won't be deleted

	// Wait for the service to be deleted - be careful to hold service
	// handle for the shortest duration possible - as this will prevent
	// it from being deleted.

	// Initial sleep interval, start fast and increase by 1s each
	// iteration to give the SCM time to remove the services.
	interval := time.Second

	// Sleep briefy before the first check, ideally this is long enough
	// for the service to be deleted on the first check.
	time.Sleep(time.Millisecond * 100)

	start := time.Now()
	for {
		s, err := m.m.OpenService(name)
		if err != nil {
			break
		}
		s.Close()
		if d := time.Since(start); d > Timeout {
			return &ServiceError{"deleting service", name, &TimeoutError{Timeout, d}}
		}
		time.Sleep(interval)
		if interval < time.Second*10 {
			interval += time.Second
		}
	}

	return nil
}

// Delete deletes the services monitored by Mgr concurrently and waits for them
// to be removed by the Service Control Manager.  Running services are stopped
// before being deleted.
func (m *Mgr) Delete() error {
	return m.iter(m.doDelete)
}

// A ServiceStatus is the status of a service.
type ServiceStatus struct {
	Name  string
	State svc.State
}

// StateString returns the string representation of the service state.
func (s *ServiceStatus) StateString() string {
	return svcStateString(s.State)
}

// Status returns the name and status for all of the services monitored.
func (m *Mgr) Status() ([]ServiceStatus, error) {
	svcs, err := m.services()
	if err != nil {
		return nil, err
	}
	defer closeServices(svcs)

	sts := make([]ServiceStatus, len(svcs))
	for i, s := range svcs {
		status, err := querySvc(s)
		if err != nil {
			return nil, err
		}
		sts[i] = ServiceStatus{Name: s.Name, State: status.State}
	}
	return sts, nil
}

func closeServices(svcs []*mgr.Service) (first error) {
	for _, s := range svcs {
		if s == nil || s.Handle == windows.InvalidHandle {
			continue
		}
		if err := s.Close(); err != nil && first == nil {
			first = err
		}
		s.Handle = windows.InvalidHandle
	}
	return
}

// Unmonitor disable start for all the Mgr m's services.
func (m *Mgr) Unmonitor() error {
	return m.iter(func(s *mgr.Service) error {
		return SetStartType(s, mgr.StartDisabled)
	})
}

// DisableAgentAutoStart sets the start type of the bosh-agent to manual.
func (m *Mgr) DisableAgentAutoStart() error {
	const name = "bosh-agent"
	s, err := m.m.OpenService("bosh-agent")
	if err != nil {
		return &ServiceError{"opening service", name, err}
	}
	defer s.Close()
	return SetStartType(s, mgr.StartManual)
}

func svcStateString(s svc.State) string {
	switch s {
	case svc.Stopped:
		return "Stopped"
	case svc.StartPending:
		return "StartPending"
	case svc.StopPending:
		return "StopPending"
	case svc.Running:
		return "Running"
	case svc.ContinuePending:
		return "ContinuePending"
	case svc.PausePending:
		return "PausePending"
	case svc.Paused:
		return "Paused"
	}
	return fmt.Sprintf("Invalid Service State: %d", s)
}
