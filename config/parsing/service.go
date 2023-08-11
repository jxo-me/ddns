package parsing

import (
	"errors"
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/core/ddns"
	"github.com/jxo-me/ddns/core/logger"
	"github.com/jxo-me/ddns/core/service"
	"github.com/jxo-me/ddns/x/ddns/alidns"
	"github.com/jxo-me/ddns/x/ddns/baidu"
	"github.com/jxo-me/ddns/x/ddns/callback"
	"github.com/jxo-me/ddns/x/ddns/cloudflare"
	"github.com/jxo-me/ddns/x/ddns/dnspod"
	"github.com/jxo-me/ddns/x/ddns/godaddy"
	"github.com/jxo-me/ddns/x/ddns/google"
	"github.com/jxo-me/ddns/x/ddns/huawei"
	"github.com/jxo-me/ddns/x/ddns/namecheap"
	"github.com/jxo-me/ddns/x/ddns/namesilo"
	"github.com/jxo-me/ddns/x/ddns/porkbun"
	"github.com/jxo-me/ddns/x/ddns/tencent"
	xservice "github.com/jxo-me/ddns/x/service"
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

func ParseService(cfg *config.DnsConfig, log logger.ILogger) (service.IDDNSService, error) {
	var dns ddns.IDDNS
	for name, iddns := range DDNS {
		if cfg.Name == name {
			dns = iddns
		}
	}
	if dns == nil {
		return nil, ErrDnsNotSupported
	}
	s := xservice.NewDDNS(dns, log, cfg)
	return s, nil
}
