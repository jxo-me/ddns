package ddns

import (
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/consts"
	"github.com/jxo-me/ddns/core/cache"
	"github.com/jxo-me/ddns/core/logger"
	"net/url"
	"strings"
)

// 固定的主域名
var staticMainDomains = []string{"com.cn", "org.cn", "net.cn", "ac.cn", "eu.org"}

// Domains Ipv4/Ipv6 domains
type Domains struct {
	Ipv4Addr    string
	Ipv4Cache   cache.IIpCache
	Ipv4Domains []*Domain
	Ipv6Addr    string
	Ipv6Cache   cache.IIpCache
	Ipv6Domains []*Domain
	Logger      logger.ILogger
}

// GetNewIp 接口/网卡/命令获得 ip 并校验用户输入的域名
func (domains *Domains) GetNewIp(dnsConf *config.DDnsConfig) {
	domains.Ipv4Domains = checkParseDomains(dnsConf.Ipv4.Domains, domains.Logger)
	domains.Ipv6Domains = checkParseDomains(dnsConf.Ipv6.Domains, domains.Logger)

	// IPv4
	if dnsConf.Ipv4.Enable && len(domains.Ipv4Domains) > 0 {
		ipv4Addr := dnsConf.GetIpv4Addr()
		if ipv4Addr != "" {
			domains.Ipv4Addr = ipv4Addr
			domains.Ipv4Cache.ResetFailedTimes()
		} else {
			// 启用IPv4 & 未获取到IP & 填写了域名 & 失败刚好3次，防止偶尔的网络连接失败，并且只发一次
			domains.Ipv4Cache.IncreaseFailedTimes()
			if domains.Ipv4Cache.GetFailedTimes() == 3 {
				domains.Ipv4Domains[0].UpdateStatus = consts.UpdatedFailed
			}
			domains.Logger.Info("Failed to obtain IPv4 address, will not update")
		}
	}

	// IPv6
	if dnsConf.Ipv6.Enable && len(domains.Ipv6Domains) > 0 {
		ipv6Addr := dnsConf.GetIpv6Addr()
		if ipv6Addr != "" {
			domains.Ipv6Addr = ipv6Addr
			domains.Ipv6Cache.ResetFailedTimes()
		} else {
			// 启用IPv6 & 未获取到IP & 填写了域名 & 失败刚好3次，防止偶尔的网络连接失败，并且只发一次
			domains.Ipv6Cache.IncreaseFailedTimes()
			if domains.Ipv6Cache.GetFailedTimes() == 3 {
				domains.Ipv6Domains[0].UpdateStatus = consts.UpdatedFailed
			}
			domains.Logger.Info("Failed to obtain IPv6 address, will not update")
		}
	}

}

// checkParseDomains 校验并解析用户输入的域名
func checkParseDomains(domainArr []string, log logger.ILogger) (domains []*Domain) {
	for _, domainStr := range domainArr {
		domainStr = strings.TrimSpace(domainStr)
		if domainStr != "" {
			domain := &Domain{}

			dp := strings.Split(domainStr, ":")
			dplen := len(dp)
			if dplen == 1 { // 自动识别域名
				sp := strings.Split(domainStr, ".")
				length := len(sp)
				if length <= 1 {
					log.Info(domainStr, "Incorrect domain name")
					continue
				}
				// 处理域名
				domain.DomainName = sp[length-2] + "." + sp[length-1]
				// 如包含在org.cn等顶级域名下，后三个才为用户主域名
				for _, staticMainDomain := range staticMainDomains {
					// 移除 domain.DomainName 的查询字符串以便与 staticMainDomain 进行比较。
					// 查询字符串是 URL ? 后面的部分。
					// 查询字符串的存在会导致顶级域名无法与 staticMainDomain 精确匹配，从而被误认为二级域名。
					// 示例："com.cn?param=value" 将被替换为 "com.cn"。
					if staticMainDomain == strings.Split(domain.DomainName, "?")[0] {
						domain.DomainName = sp[length-3] + "." + domain.DomainName
						break
					}
				}

				domainLen := len(domainStr) - len(domain.DomainName)
				if domainLen > 0 {
					domain.SubDomain = domainStr[:domainLen-1]
				} else {
					domain.SubDomain = domainStr[:domainLen]
				}

			} else if dplen == 2 { // 主机记录:域名 格式
				sp := strings.Split(dp[1], ".")
				length := len(sp)
				if length <= 1 {
					log.Info(domainStr, "Incorrect domain name")
					continue
				}
				domain.DomainName = dp[1]
				domain.SubDomain = dp[0]
			} else {
				log.Info(domainStr, "Incorrect domain name")
				continue
			}

			// 参数条件
			if strings.Contains(domain.DomainName, "?") {
				u, err := url.Parse("http://" + domain.DomainName)
				if err != nil {
					log.Info(domainStr, "domain name resolution failed")
					continue
				}
				domain.DomainName = u.Host
				domain.CustomParams = u.Query().Encode()
			}
			domains = append(domains, domain)
		}
	}
	return
}

// GetNewIpResult 获得GetNewIp结果
func (domains *Domains) GetNewIpResult(recordType string) (ipAddr string, retDomains []*Domain) {
	if recordType == "AAAA" {
		if domains.Ipv6Cache.Check(domains.Ipv6Addr) {
			return domains.Ipv6Addr, domains.Ipv6Domains
		} else {
			domains.Logger.Infof("IPv6 has not changed, will wait %d times before comparing with DNS service provider\n", domains.Ipv6Cache.GetTimes())
			return "", domains.Ipv6Domains
		}
	}
	// IPv4
	if domains.Ipv4Cache.Check(domains.Ipv4Addr) {
		return domains.Ipv4Addr, domains.Ipv4Domains
	} else {
		domains.Logger.Infof("IPv4 has not changed, will wait %d times before comparing with DNS service provider\n", domains.Ipv4Cache.GetTimes())
		return "", domains.Ipv4Domains
	}
}
