package cloudflare

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
	Endpoint string = "https://api.cloudflare.com/client/v4/zones"
	Code     string = "cloudflare"
)

// Cloudflare Cloudflare实现
type Cloudflare struct {
	DNS     *config.DNS
	Domains ddns.Domains
	TTL     int
	logger  logger.ILogger
}

// CloudflareZonesResp cloudflare zones返回结果
type CloudflareZonesResp struct {
	CloudflareStatus
	Result []struct {
		ID     string
		Name   string
		Status string
		Paused bool
	}
}

// CloudflareRecordsResp records
type CloudflareRecordsResp struct {
	CloudflareStatus
	Result []CloudflareRecord
}

// CloudflareRecord 记录实体
type CloudflareRecord struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	Proxied bool   `json:"proxied"`
	TTL     int    `json:"ttl"`
}

// CloudflareStatus 公共状态
type CloudflareStatus struct {
	Success  bool
	Messages []string
}

func (cf *Cloudflare) String() string {
	return Code
}

func (cf *Cloudflare) Endpoint() string {
	return Endpoint
}

// Init 初始化
func (cf *Cloudflare) Init(dnsConf *config.DDnsConfig, ipv4cache cache.IIpCache, ipv6cache cache.IIpCache, log logger.ILogger) {
	cf.Domains.Ipv4Cache = ipv4cache
	cf.Domains.Ipv6Cache = ipv6cache
	cf.Domains.Logger = log
	cf.DNS = dnsConf.DNS
	cf.Domains.GetNewIp(dnsConf)
	cf.logger = log
	if dnsConf.TTL == "" {
		// 默认1 auto ttl
		cf.TTL = 1
	} else {
		ttl, err := strconv.Atoi(dnsConf.TTL)
		if err != nil {
			cf.TTL = 1
		} else {
			cf.TTL = ttl
		}
	}
}

// AddUpdateDomainRecords 添加或更新IPv4/IPv6记录
func (cf *Cloudflare) AddUpdateDomainRecords() ddns.Domains {
	cf.addUpdateDomainRecords("A")
	cf.addUpdateDomainRecords("AAAA")
	return cf.Domains
}

func (cf *Cloudflare) addUpdateDomainRecords(recordType string) {
	ipAddr, domains := cf.Domains.GetNewIpResult(recordType)

	if ipAddr == "" {
		return
	}

	for _, domain := range domains {
		// get zone
		result, err := cf.getZones(domain)
		if err != nil || len(result.Result) != 1 {
			domain.UpdateStatus = consts.UpdatedFailed
			return
		}
		zoneID := result.Result[0].ID

		var records CloudflareRecordsResp
		// getDomains 最多更新前50条
		err = cf.request(
			"GET",
			fmt.Sprintf(Endpoint+"/%s/dns_records?type=%s&name=%s&per_page=50", zoneID, recordType, domain),
			nil,
			&records,
		)

		if err != nil || !records.Success {
			return
		}

		if len(records.Result) > 0 {
			// 更新
			cf.modify(records, zoneID, domain, recordType, ipAddr)
		} else {
			// 新增
			cf.create(zoneID, domain, recordType, ipAddr)
		}
	}
}

// 创建
func (cf *Cloudflare) create(zoneID string, domain *ddns.Domain, recordType string, ipAddr string) {
	record := &CloudflareRecord{
		Type:    recordType,
		Name:    domain.String(),
		Content: ipAddr,
		Proxied: false,
		TTL:     cf.TTL,
	}
	record.Proxied = domain.GetCustomParams().Get("proxied") == "true"
	var status CloudflareStatus
	err := cf.request(
		"POST",
		fmt.Sprintf(Endpoint+"/%s/dns_records", zoneID),
		record,
		&status,
	)
	if err == nil && status.Success {
		cf.logger.Infof("新增域名解析 %s 成功！IP: %s", domain, ipAddr)
		domain.UpdateStatus = consts.UpdatedSuccess
	} else {
		cf.logger.Infof("新增域名解析 %s 失败！Messages: %s", domain, status.Messages)
		domain.UpdateStatus = consts.UpdatedFailed
	}
}

// 修改
func (cf *Cloudflare) modify(result CloudflareRecordsResp, zoneID string, domain *ddns.Domain, recordType string, ipAddr string) {
	for _, record := range result.Result {
		// 相同不修改
		if record.Content == ipAddr {
			cf.logger.Infof("你的IP %s 没有变化, 域名 %s", ipAddr, domain)
			continue
		}
		var status CloudflareStatus
		record.Content = ipAddr
		record.TTL = cf.TTL
		// 存在参数才修改proxied
		if domain.GetCustomParams().Has("proxied") {
			record.Proxied = domain.GetCustomParams().Get("proxied") == "true"
		}
		err := cf.request(
			"PUT",
			fmt.Sprintf(Endpoint+"/%s/dns_records/%s", zoneID, record.ID),
			record,
			&status,
		)
		if err == nil && status.Success {
			cf.logger.Infof("更新域名解析 %s 成功！IP: %s", domain, ipAddr)
			domain.UpdateStatus = consts.UpdatedSuccess
		} else {
			cf.logger.Infof("更新域名解析 %s 失败！Messages: %s", domain, status.Messages)
			domain.UpdateStatus = consts.UpdatedFailed
		}
	}
}

// 获得域名记录列表
func (cf *Cloudflare) getZones(domain *ddns.Domain) (result CloudflareZonesResp, err error) {
	err = cf.request(
		"GET",
		fmt.Sprintf(Endpoint+"?name=%s&status=%s&per_page=%s", domain.DomainName, "active", "50"),
		nil,
		&result,
	)

	return
}

// request 统一请求接口
func (cf *Cloudflare) request(method string, url string, data interface{}, result interface{}) (err error) {
	jsonStr := make([]byte, 0)
	if data != nil {
		jsonStr, _ = json.Marshal(data)
	}
	req, err := http.NewRequest(
		method,
		url,
		bytes.NewBuffer(jsonStr),
	)
	if err != nil {
		cf.logger.Infof("http.NewRequest失败. Error: ", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+cf.DNS.Secret)
	req.Header.Set("Content-Type", "application/json")

	client := util.CreateHTTPClient()
	resp, err := client.Do(req)
	err = util.GetHTTPResponse(resp, url, err, result)

	return
}
