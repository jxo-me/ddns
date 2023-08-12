package hook

import (
	"github.com/jxo-me/ddns/consts"
	"github.com/jxo-me/ddns/x/ddns"
)

type IHook interface {
	String() string
	ExecHook(domains *ddns.Domains) (consts.UpdateStatusType, consts.UpdateStatusType)
}
