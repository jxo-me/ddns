package parsing

import (
	"errors"
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/core/ddns"
	"github.com/jxo-me/ddns/core/logger"
	"github.com/jxo-me/ddns/core/service"
	"github.com/jxo-me/ddns/sdk/ddns/alidns"
	"github.com/jxo-me/ddns/sdk/ddns/baidu"
	"github.com/jxo-me/ddns/sdk/ddns/callback"
	"github.com/jxo-me/ddns/sdk/ddns/cloudflare"
	"github.com/jxo-me/ddns/sdk/ddns/dnspod"
	"github.com/jxo-me/ddns/sdk/ddns/godaddy"
	"github.com/jxo-me/ddns/sdk/ddns/google"
	"github.com/jxo-me/ddns/sdk/ddns/huawei"
	"github.com/jxo-me/ddns/sdk/ddns/namecheap"
	"github.com/jxo-me/ddns/sdk/ddns/namesilo"
	"github.com/jxo-me/ddns/sdk/ddns/porkbun"
	"github.com/jxo-me/ddns/sdk/ddns/tencent"
	xservice "github.com/jxo-me/ddns/sdk/service"
)

var (
	ErrDnsNotSupported = errors.New("dns not supported")
	DDNS               = map[string]ddns.IDDNS{
		alidns.Code:     &alidns.Alidns{},
		baidu.Code:      &baidu.BaiduCloud{},
		callback.Code:   &callback.Callback{},
		cloudflare.Code: &cloudflare.Cloudflare{},
		dnspod.Code:     &dnspod.Dnspod{},
		godaddy.Code:    &godaddy.GoDaddyDNS{},
		google.Code:     &google.GoogleDomain{},
		huawei.Code:     &huawei.Huaweicloud{},
		namecheap.Code:  &namecheap.NameCheap{},
		namesilo.Code:   &namesilo.NameSilo{},
		porkbun.Code:    &porkbun.Porkbun{},
		tencent.Code:    &tencent.TencentCloud{},
	}
)

func ParseService(cfg *config.DDnsConfig, log logger.ILogger) (service.IDDNSService, error) {
	var dns ddns.IDDNS
	for name, iddns := range DDNS {
		if cfg.Name == name {
			dns = iddns
		}
	}
	if dns == nil {
		return nil, ErrDnsNotSupported
	}
	s := xservice.NewDDNSService(dns, log, cfg)
	return s, nil
}
