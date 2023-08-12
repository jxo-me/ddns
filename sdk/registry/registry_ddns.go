package registry

import "github.com/jxo-me/ddns/core/service"

type DDNSRegistry struct {
	registry[service.IDDNSService]
}
