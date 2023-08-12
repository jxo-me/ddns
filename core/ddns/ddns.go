package ddns

import (
	"github.com/jxo-me/ddns/cache"
	"github.com/jxo-me/ddns/config"
)

// IDDNS interface
type IDDNS interface {
	String() string
	// Endpoint GetEndpoint
	Endpoint() string
	Init(dnsConf *config.DDnsConfig, ipv4cache *cache.IpCache, ipv6cache *cache.IpCache)
	// AddUpdateDomainRecords 添加或更新IPv4/IPv6记录
	AddUpdateDomainRecords() (domains config.Domains)
}
