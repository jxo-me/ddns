package overwatch

import "github.com/jxo-me/ddns/core/service"

// ServiceCallback is a service notify it's run loop finished.
// the first parameter is the service type,
// the second parameter is the service name,
// the third parameter is an optional error if the service failed
type ServiceCallback func(string, string, error)

// AppManager is the default implementation of over-watched service management
type AppManager struct {
	services map[string]service.IDDNSService
	callback ServiceCallback
}

// NewAppManager creates a new over-watched manager
func NewAppManager(callback ServiceCallback) Manager {
	return &AppManager{services: make(map[string]service.IDDNSService), callback: callback}
}

// Add takes in a new service to manage.
// It stops the service if it already exists in the manager and is running
// It then starts the newly added service
func (m *AppManager) Add(service service.IDDNSService) {
	// check for existing service
	if currentService, ok := m.services[service.String()]; ok {
		if currentService.Hash() == service.Hash() {
			return // the exact same service, no changes, so move along
		}
		currentService.Stop() //shutdown the listener since a new one is starting
	}
	m.services[service.String()] = service

	//start the service!
	go m.serviceRun(service)
}

// Remove shutdowns the service by name and removes it from its current management list
func (m *AppManager) Remove(name string) {
	if currentService, ok := m.services[name]; ok {
		_ = currentService.Stop()
	}
	delete(m.services, name)
}

// Services returns all the current Services being managed
func (m *AppManager) Services() []service.IDDNSService {
	var values []service.IDDNSService
	for _, value := range m.services {
		values = append(values, value)
	}
	return values
}

func (m *AppManager) serviceRun(service service.IDDNSService) {
	err := service.Start()
	if m.callback != nil {
		m.callback(service.String(), service.String(), err)
	}
}
