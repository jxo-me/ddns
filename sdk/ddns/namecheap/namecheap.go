package namecheap

import (
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/consts"
	"github.com/jxo-me/ddns/core/cache"
	"github.com/jxo-me/ddns/core/logger"
	"github.com/jxo-me/ddns/internal/util"
	"github.com/jxo-me/ddns/sdk/ddns"
	"io"
	"net/http"
	"strings"
)

const (
	Endpoint string = "https://dynamicdns.park-your-domain.com/update?host=#{host}&domain=#{domain}&password=#{password}&ip=#{ip}"
	Code     string = "namecheap"
)

// NameCheap Domain
type NameCheap struct {
	DNS      *config.DNS
	Domains  ddns.Domains
	lastIpv4 string
	lastIpv6 string
	logger   logger.ILogger
}

// NameCheapResp 修改域名解析结果
type NameCheapResp struct {
	Status string
	Errors []string
}

func (nc *NameCheap) String() string {
	return Code
}

func (nc *NameCheap) Endpoint() string {
	return Endpoint
}

// Init 初始化
func (nc *NameCheap) Init(dnsConf *config.DDnsConfig, ipv4cache cache.IIpCache, ipv6cache cache.IIpCache, log logger.ILogger) {
	nc.Domains.Ipv4Cache = ipv4cache
	nc.Domains.Ipv6Cache = ipv6cache
	nc.Domains.Logger = log
	nc.lastIpv4 = ipv4cache.GetAddr()
	nc.lastIpv6 = ipv6cache.GetAddr()

	nc.DNS = dnsConf.DNS
	nc.Domains.GetNewIp(dnsConf)
	nc.logger = log
}

// AddUpdateDomainRecords 添加或更新IPv4/IPv6记录
func (nc *NameCheap) AddUpdateDomainRecords() ddns.Domains {
	nc.addUpdateDomainRecords("A")
	nc.addUpdateDomainRecords("AAAA")
	return nc.Domains
}

func (nc *NameCheap) addUpdateDomainRecords(recordType string) {
	ipAddr, domains := nc.Domains.GetNewIpResult(recordType)

	if ipAddr == "" {
		return
	}

	// 防止多次发送Webhook通知
	if recordType == "A" {
		if nc.lastIpv4 == ipAddr {
			nc.logger.Infof("你的IPv4未变化, 未触发Namecheap请求")
			return
		}
	} else {
		// https://www.namecheap.com/support/knowledgebase/article.aspx/29/11/how-to-dynamically-update-the-hosts-ip-with-an-http-request/
		nc.logger.Infof("Namecheap DDNS 不支持更新 IPv6！")
		return
		// if nc.lastIpv6 == ipAddr {
		// 	nc.logger.Infof("你的IPv6未变化, 未触发Namecheap请求")
		// 	return
		// }
	}

	for _, domain := range domains {
		nc.modify(domain, recordType, ipAddr)
	}
}

// 修改
func (nc *NameCheap) modify(domain *ddns.Domain, recordType string, ipAddr string) {
	var result NameCheapResp
	err := nc.request(&result, ipAddr, domain)

	if err != nil {
		nc.logger.Infof("修改域名解析 %s 失败！", domain)
		domain.UpdateStatus = consts.UpdatedFailed
		return
	}

	switch result.Status {
	case "Success":
		nc.logger.Infof("修改域名解析 %s 成功！IP: %s\n", domain, ipAddr)
		domain.UpdateStatus = consts.UpdatedSuccess
	default:
		nc.logger.Infof("修改域名解析 %s 失败！Status: %s\n", domain, result.Status)
		domain.UpdateStatus = consts.UpdatedFailed
	}
}

// request 统一请求接口
func (nc *NameCheap) request(result *NameCheapResp, ipAddr string, domain *ddns.Domain) (err error) {
	var url string = Endpoint
	url = strings.ReplaceAll(url, "#{host}", domain.GetSubDomain())
	url = strings.ReplaceAll(url, "#{domain}", domain.DomainName)
	url = strings.ReplaceAll(url, "#{password}", nc.DNS.Secret)
	url = strings.ReplaceAll(url, "#{ip}", ipAddr)

	req, err := http.NewRequest(
		http.MethodGet,
		url,
		http.NoBody,
	)

	if err != nil {
		nc.logger.Infof("http.NewRequest失败. Error: ", err)
		return
	}

	client := util.CreateHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		nc.logger.Infof("client.Do失败. Error: ", err)
		return
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		nc.logger.Infof("请求namecheap失败")
		return err
	}

	status := string(data)

	if strings.Contains(status, "<ErrCount>0</ErrCount>") {
		result.Status = "Success"
	} else {
		result.Status = status
	}

	return
}
