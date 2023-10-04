package overwatch

import "github.com/jxo-me/ddns/core/service"

// Manager is based type to manage running services
type Manager interface {
	Add(service service.IDDNSService)
	Remove(string)
	Services() []service.IDDNSService
}
