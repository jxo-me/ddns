package google

import (
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/consts"
	"github.com/jxo-me/ddns/core/cache"
	"github.com/jxo-me/ddns/core/logger"
	"github.com/jxo-me/ddns/internal/util"
	"github.com/jxo-me/ddns/sdk/ddns"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	Endpoint string = "https://domains.google.com/nic/update"
	Code     string = "google"
)

// GoogleDomain Google Domain
// https://support.google.com/domains/answer/6147083?hl=zh-Hans#zippy=%2C使用-api-更新您的动态-dns-记录
type GoogleDomain struct {
	DNS      *config.DNS
	Domains  ddns.Domains
	lastIpv4 string
	lastIpv6 string
	logger   logger.ILogger
}

// GoogleDomainResp 修改域名解析结果
type GoogleDomainResp struct {
	Status  string
	SetedIP string
}

func (gd *GoogleDomain) Endpoint() string {
	return Endpoint
}

// Init 初始化
func (gd *GoogleDomain) Init(dnsConf *config.DDnsConfig, ipv4cache cache.IIpCache, ipv6cache cache.IIpCache, log logger.ILogger) {
	gd.Domains.Ipv4Cache = ipv4cache
	gd.Domains.Ipv6Cache = ipv6cache
	gd.Domains.Logger = log
	gd.DNS = dnsConf.DNS
	gd.Domains.GetNewIp(dnsConf)
	gd.logger = log
}

// AddUpdateDomainRecords 添加或更新IPv4/IPv6记录
func (gd *GoogleDomain) AddUpdateDomainRecords() ddns.Domains {
	gd.addUpdateDomainRecords("A")
	gd.addUpdateDomainRecords("AAAA")
	return gd.Domains
}

func (gd *GoogleDomain) addUpdateDomainRecords(recordType string) {
	ipAddr, domains := gd.Domains.GetNewIpResult(recordType)

	if ipAddr == "" {
		return
	}

	// 防止多次发送Webhook通知
	if recordType == "A" {
		if gd.lastIpv4 == ipAddr {
			gd.logger.Infof("你的IPv4未变化, 未触发Google请求")
			return
		}
	} else {
		if gd.lastIpv6 == ipAddr {
			gd.logger.Infof("你的IPv6未变化, 未触发Google请求")
			return
		}
	}

	for _, domain := range domains {
		gd.modify(domain, recordType, ipAddr)
	}
}

func (gd *GoogleDomain) String() string {
	return Code
}

// 修改
func (gd *GoogleDomain) modify(domain *ddns.Domain, recordType string, ipAddr string) {
	params := domain.GetCustomParams()
	params.Set("hostname", domain.GetFullDomain())
	params.Set("myip", ipAddr)

	var result GoogleDomainResp
	err := gd.request(params, &result)

	if err != nil {
		gd.logger.Infof("修改域名解析 %s 失败！", domain)
		domain.UpdateStatus = consts.UpdatedFailed
		return
	}

	switch result.Status {
	case "nochg":
		gd.logger.Infof("你的IP %s 没有变化, 域名 %s", ipAddr, domain)
	case "good":
		gd.logger.Infof("修改域名解析 %s 成功！IP: %s", domain, ipAddr)
		domain.UpdateStatus = consts.UpdatedSuccess
	default:
		gd.logger.Infof("修改域名解析 %s 失败！Status: %s", domain, result.Status)
		domain.UpdateStatus = consts.UpdatedFailed
	}
}

// request 统一请求接口
func (gd *GoogleDomain) request(params url.Values, result *GoogleDomainResp) (err error) {

	req, err := http.NewRequest(
		http.MethodPost,
		Endpoint,
		http.NoBody,
	)

	if err != nil {
		gd.logger.Infof("http.NewRequest失败. Error: ", err)
		return
	}

	req.URL.RawQuery = params.Encode()
	req.SetBasicAuth(gd.DNS.ID, gd.DNS.Secret)

	client := util.CreateHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		gd.logger.Infof("client.Do失败. Error: ", err)
		return
	}

	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	status := string(data)

	if s := strings.Split(status, " "); s[0] == "good" || s[0] == "nochg" { // Success status
		result.Status = s[0]
		if len(s) > 1 {
			result.SetedIP = s[1]
		}
	} else {
		result.Status = status
	}
	return
}
