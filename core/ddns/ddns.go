package ddns

import (
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/core/cache"
	"github.com/jxo-me/ddns/sdk/ddns"
)

// IDDNS interface
type IDDNS interface {
	String() string
	// Endpoint GetEndpoint
	Endpoint() string
	Init(dnsConf *config.DDnsConfig, ipv4cache cache.IIpCache, ipv6cache cache.IIpCache)
	// AddUpdateDomainRecords 添加或更新IPv4/IPv6记录
	AddUpdateDomainRecords() (domains ddns.Domains)
}
