package callback

import (
	"encoding/json"
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/consts"
	"github.com/jxo-me/ddns/core/cache"
	"github.com/jxo-me/ddns/core/logger"
	"github.com/jxo-me/ddns/internal/util"
	"github.com/jxo-me/ddns/sdk/ddns"
	"net/http"
	"net/url"
	"strings"
)

const (
	Code = "callback"
)

type Callback struct {
	DNS      *config.DNS
	Domains  ddns.Domains
	TTL      string
	lastIpv4 string
	lastIpv6 string
	logger   logger.ILogger
}

func (cb *Callback) String() string {
	return Code
}

func (cb *Callback) Endpoint() string {
	return ""
}

// Init 初始化
func (cb *Callback) Init(dnsConf *config.DDnsConfig, ipv4cache cache.IIpCache, ipv6cache cache.IIpCache, log logger.ILogger) {
	cb.Domains.Ipv4Cache = ipv4cache
	cb.Domains.Ipv6Cache = ipv6cache
	cb.lastIpv4 = ipv4cache.GetAddr()
	cb.lastIpv6 = ipv6cache.GetAddr()

	cb.DNS = dnsConf.DNS
	cb.Domains.GetNewIp(dnsConf)
	cb.logger = log
	if dnsConf.TTL == "" {
		// 默认600
		cb.TTL = "600"
	} else {
		cb.TTL = dnsConf.TTL
	}
}

// AddUpdateDomainRecords 添加或更新IPv4/IPv6记录
func (cb *Callback) AddUpdateDomainRecords() ddns.Domains {
	cb.addUpdateDomainRecords("A")
	cb.addUpdateDomainRecords("AAAA")
	return cb.Domains
}

func (cb *Callback) addUpdateDomainRecords(recordType string) {
	ipAddr, domains := cb.Domains.GetNewIpResult(recordType)

	if ipAddr == "" {
		return
	}

	// 防止多次发送Webhook通知
	if recordType == "A" {
		if cb.lastIpv4 == ipAddr {
			cb.logger.Infof("你的IPv4未变化, 未触发Callback")
			return
		}
	} else {
		if cb.lastIpv6 == ipAddr {
			cb.logger.Infof("你的IPv6未变化, 未触发Callback")
			return
		}
	}

	for _, domain := range domains {
		method := "GET"
		postPara := ""
		contentType := "application/x-www-form-urlencoded"
		if cb.DNS.Secret != "" {
			method = "POST"
			postPara = replacePara(cb.DNS.Secret, ipAddr, domain, recordType, cb.TTL)
			if json.Valid([]byte(postPara)) {
				contentType = "application/json"
			}
		}
		requestURL := replacePara(cb.DNS.ID, ipAddr, domain, recordType, cb.TTL)
		u, err := url.Parse(requestURL)
		if err != nil {
			cb.logger.Infof("Callback的URL不正确")
			return
		}
		req, err := http.NewRequest(method, u.String(), strings.NewReader(postPara))
		if err != nil {
			cb.logger.Infof("创建Callback请求异常, Err:", err)
			return
		}
		req.Header.Add("content-type", contentType)

		clt := util.CreateHTTPClient()
		resp, err := clt.Do(req)
		body, err := util.GetHTTPResponseOrg(resp, requestURL, err)
		if err == nil {
			cb.logger.Infof("Callback调用成功, 域名: %s, IP: %s, 返回数据: %s, \n", domain, ipAddr, string(body))
			domain.UpdateStatus = consts.UpdatedSuccess
		} else {
			cb.logger.Infof("Callback调用失败，Err：%s\n", err)
			domain.UpdateStatus = consts.UpdatedFailed
		}
	}
}

// replacePara 替换参数
func replacePara(orgPara, ipAddr string, domain *ddns.Domain, recordType string, ttl string) (newPara string) {
	orgPara = strings.ReplaceAll(orgPara, "#{ip}", ipAddr)
	orgPara = strings.ReplaceAll(orgPara, "#{domain}", domain.String())
	orgPara = strings.ReplaceAll(orgPara, "#{recordType}", recordType)
	orgPara = strings.ReplaceAll(orgPara, "#{ttl}", ttl)

	for k, v := range domain.GetCustomParams() {
		if len(v) == 1 {
			orgPara = strings.ReplaceAll(orgPara, "#{"+k+"}", v[0])
		}
	}

	return orgPara
}
