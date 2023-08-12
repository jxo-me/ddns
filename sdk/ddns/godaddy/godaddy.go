package godaddy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/consts"
	"github.com/jxo-me/ddns/core/cache"
	"github.com/jxo-me/ddns/core/logger"
	"github.com/jxo-me/ddns/internal/util"
	"github.com/jxo-me/ddns/sdk/ddns"
	"net/http"
	"strconv"
)

const (
	Endpoint string = "https://api.godaddy.com/v1/domains"
	Code     string = "godaddy"
)

type godaddyRecord struct {
	Data string `json:"data"`
	Name string `json:"name"`
	TTL  int    `json:"ttl"`
	Type string `json:"type"`
}

type godaddyRecords []godaddyRecord

type GoDaddyDNS struct {
	dns      *config.DNS
	domains  ddns.Domains
	ttl      int
	header   http.Header
	client   *http.Client
	lastIpv4 string
	lastIpv6 string
	logger   logger.ILogger
}

func (g *GoDaddyDNS) String() string {
	return Code
}

func (g *GoDaddyDNS) Endpoint() string {
	return Endpoint
}

func (g *GoDaddyDNS) Init(dnsConf *config.DDnsConfig, ipv4cache cache.IIpCache, ipv6cache cache.IIpCache, log logger.ILogger) {
	g.domains.Ipv4Cache = ipv4cache
	g.domains.Ipv6Cache = ipv6cache
	g.domains.Logger = log
	g.lastIpv4 = ipv4cache.GetAddr()
	g.lastIpv6 = ipv6cache.GetAddr()

	g.dns = dnsConf.DNS
	g.domains.GetNewIp(dnsConf)
	g.logger = log
	g.ttl = 600
	if val, err := strconv.Atoi(dnsConf.TTL); err == nil {
		g.ttl = val
	}
	g.header = map[string][]string{
		"Authorization": {fmt.Sprintf("sso-key %s:%s", g.dns.ID, g.dns.Secret)},
		"Content-Type":  {"application/json"},
	}

	g.client = util.CreateHTTPClient()
}

func (g *GoDaddyDNS) updateDomainRecord(recordType string, ipAddr string, domains []*ddns.Domain) {
	if ipAddr == "" {
		return
	}

	// 防止多次发送Webhook通知
	if recordType == "A" {
		if g.lastIpv4 == ipAddr {
			g.logger.Infof("你的IPv4未变化, 未触发Godaddy请求")
			return
		}
	} else {
		if g.lastIpv6 == ipAddr {
			g.logger.Infof("你的IPv6未变化, 未触发Godaddy请求")
			return
		}
	}

	for _, domain := range domains {
		err := g.sendReq(http.MethodPut, recordType, domain, &godaddyRecords{godaddyRecord{
			Data: ipAddr,
			Name: domain.GetSubDomain(),
			TTL:  g.ttl,
			Type: recordType,
		}})
		if err == nil {
			g.logger.Infof("更新域名解析 %s 成功! IP: %s", domain, ipAddr)
			domain.UpdateStatus = consts.UpdatedSuccess
		} else {
			g.logger.Infof("更新域名解析 %s 失败！", domain)
			domain.UpdateStatus = consts.UpdatedFailed
		}
	}
}

func (g *GoDaddyDNS) AddUpdateDomainRecords() ddns.Domains {
	if ipv4Addr, ipv4Domains := g.domains.GetNewIpResult("A"); ipv4Addr != "" {
		g.updateDomainRecord("A", ipv4Addr, ipv4Domains)
	}
	if ipv6Addr, ipv6Domains := g.domains.GetNewIpResult("AAAA"); ipv6Addr != "" {
		g.updateDomainRecord("AAAA", ipv6Addr, ipv6Domains)
	}
	return g.domains
}

func (g *GoDaddyDNS) sendReq(method string, rType string, domain *ddns.Domain, data *godaddyRecords) error {

	var body *bytes.Buffer
	if data != nil {
		if buffer, err := json.Marshal(data); err != nil {
			return err
		} else {
			body = bytes.NewBuffer(buffer)
		}
	}
	path := fmt.Sprintf("%s/%s/records/%s/%s", Endpoint, domain.DomainName, rType, domain.GetSubDomain())

	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return err
	}
	req.Header = g.header
	resp, err := g.client.Do(req)
	_, err = util.GetHTTPResponseOrg(resp, path, err)
	return err
}
