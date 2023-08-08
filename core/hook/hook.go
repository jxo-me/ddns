package hook

import (
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/consts"
)

type IHook interface {
	String() string
	ExecHook(domains *config.Domains) (consts.UpdateStatusType, consts.UpdateStatusType)
}
