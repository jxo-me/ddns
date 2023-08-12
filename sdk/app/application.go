package app

import (
	"github.com/jxo-me/ddns/core/app"
	reg "github.com/jxo-me/ddns/core/registry"
	"github.com/jxo-me/ddns/core/service"
	"github.com/jxo-me/ddns/sdk/registry"
)

var (
	Runtime app.IRuntime = NewConfig()
)

type Application struct {
	ddnsReg reg.IRegistry[service.IDDNSService]
}

func NewConfig() *Application {
	a := Application{
		ddnsReg: new(registry.DDNSRegistry),
	}

	return &a
}

func (a *Application) DDNSRegistry() reg.IRegistry[service.IDDNSService] {
	return a.ddnsReg
}
