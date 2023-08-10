package app

import (
	reg "github.com/jxo-me/ddns/core/registry"
	"github.com/jxo-me/ddns/core/service"
)

type IRuntime interface {
	DDNSRegistry() reg.IRegistry[service.IDDNS]
}
