package ddns

import (
	"github.com/jxo-me/ddns/cache"
	"github.com/jxo-me/ddns/config"
)

// IDDNS interface
type IDDNS interface {
	String() string
	// GetEndpoint 获取DDNS服务端点
	Endpoint() string
	Init(dnsConf *config.DnsConfig, ipv4cache *cache.IpCache, ipv6cache *cache.IpCache)
	// AddUpdateDomainRecords 添加或更新IPv4/IPv6记录
	AddUpdateDomainRecords() (domains config.Domains)
}
