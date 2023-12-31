package huawei

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
	Endpoint string = "https://dns.myhuaweicloud.com"
	Code     string = "huaweicloud"
)

// Huaweicloud Huaweicloud
// https://support.huaweicloud.com/api-dns/dns_api_64001.html
type Huaweicloud struct {
	DNS     *config.DNS
	Domains ddns.Domains
	TTL     int
	logger  logger.ILogger
}

// HuaweicloudZonesResp zones response
type HuaweicloudZonesResp struct {
	Zones []struct {
		ID         string
		Name       string
		Recordsets []HuaweicloudRecordsets
	}
}

// HuaweicloudRecordsResp 记录返回结果
type HuaweicloudRecordsResp struct {
	Recordsets []HuaweicloudRecordsets
}

// HuaweicloudRecordsets 记录
type HuaweicloudRecordsets struct {
	ID      string
	Name    string `json:"name"`
	ZoneID  string `json:"zone_id"`
	Status  string
	Type    string   `json:"type"`
	TTL     int      `json:"ttl"`
	Records []string `json:"records"`
}

func (hw *Huaweicloud) String() string {
	return Code
}

func (hw *Huaweicloud) Endpoint() string {
	return Endpoint
}

// Init 初始化
func (hw *Huaweicloud) Init(dnsConf *config.DDnsConfig, ipv4cache cache.IIpCache, ipv6cache cache.IIpCache, log logger.ILogger) {
	hw.Domains.Ipv4Cache = ipv4cache
	hw.Domains.Ipv6Cache = ipv6cache
	hw.Domains.Logger = log
	hw.DNS = dnsConf.DNS
	hw.Domains.GetNewIp(dnsConf)
	hw.logger = log
	if dnsConf.TTL == "" {
		// 默认300s
		hw.TTL = 300
	} else {
		ttl, err := strconv.Atoi(dnsConf.TTL)
		if err != nil {
			hw.TTL = 300
		} else {
			hw.TTL = ttl
		}
	}
}

// AddUpdateDomainRecords 添加或更新IPv4/IPv6记录
func (hw *Huaweicloud) AddUpdateDomainRecords() ddns.Domains {
	hw.addUpdateDomainRecords("A")
	hw.addUpdateDomainRecords("AAAA")
	return hw.Domains
}

func (hw *Huaweicloud) addUpdateDomainRecords(recordType string) {
	ipAddr, domains := hw.Domains.GetNewIpResult(recordType)

	if ipAddr == "" {
		return
	}

	for _, domain := range domains {

		var records HuaweicloudRecordsResp

		err := hw.request(
			"GET",
			fmt.Sprintf(Endpoint+"/v2/recordsets?type=%s&name=%s", recordType, domain),
			nil,
			&records,
		)

		if err != nil {
			domain.UpdateStatus = consts.UpdatedFailed
			return
		}

		find := false
		for _, record := range records.Recordsets {
			// 名称相同才更新。华为云默认是模糊搜索
			if record.Name == domain.String()+"." {
				// 更新
				hw.modify(record, domain, recordType, ipAddr)
				find = true
				break
			}
		}

		if !find {
			// 新增
			hw.create(domain, recordType, ipAddr)
		}

	}
}

// 创建
func (hw *Huaweicloud) create(domain *ddns.Domain, recordType string, ipAddr string) {
	zone, err := hw.getZones(domain)
	if err != nil {
		return
	}
	if len(zone.Zones) == 0 {
		hw.logger.Infof("未能找到公网域名, 请检查域名是否添加")
		return
	}

	zoneID := zone.Zones[0].ID
	for _, z := range zone.Zones {
		if z.Name == domain.DomainName+"." {
			zoneID = z.ID
			break
		}
	}

	record := &HuaweicloudRecordsets{
		Type:    recordType,
		Name:    domain.String() + ".",
		Records: []string{ipAddr},
		TTL:     hw.TTL,
	}
	var result HuaweicloudRecordsets
	err = hw.request(
		"POST",
		fmt.Sprintf(Endpoint+"/v2/zones/%s/recordsets", zoneID),
		record,
		&result,
	)
	if err == nil && (len(result.Records) > 0 && result.Records[0] == ipAddr) {
		hw.logger.Infof("新增域名解析 %s 成功！IP: %s", domain, ipAddr)
		domain.UpdateStatus = consts.UpdatedSuccess
	} else {
		hw.logger.Infof("新增域名解析 %s 失败！Status: %s", domain, result.Status)
		domain.UpdateStatus = consts.UpdatedFailed
	}
}

// 修改
func (hw *Huaweicloud) modify(record HuaweicloudRecordsets, domain *ddns.Domain, recordType string, ipAddr string) {

	// 相同不修改
	if len(record.Records) > 0 && record.Records[0] == ipAddr {
		hw.logger.Infof("你的IP %s 没有变化, 域名 %s", ipAddr, domain)
		return
	}

	var request map[string]interface{} = make(map[string]interface{})
	request["records"] = []string{ipAddr}
	request["ttl"] = hw.TTL

	var result HuaweicloudRecordsets

	err := hw.request(
		"PUT",
		fmt.Sprintf(Endpoint+"/v2/zones/%s/recordsets/%s", record.ZoneID, record.ID),
		&request,
		&result,
	)

	if err == nil && (len(result.Records) > 0 && result.Records[0] == ipAddr) {
		hw.logger.Infof("更新域名解析 %s 成功！IP: %s, 状态: %s", domain, ipAddr, result.Status)
		domain.UpdateStatus = consts.UpdatedSuccess
	} else {
		hw.logger.Infof("更新域名解析 %s 失败！Status: %s", domain, result.Status)
		domain.UpdateStatus = consts.UpdatedFailed
	}
}

// 获得域名记录列表
func (hw *Huaweicloud) getZones(domain *ddns.Domain) (result HuaweicloudZonesResp, err error) {
	err = hw.request(
		"GET",
		fmt.Sprintf(Endpoint+"/v2/zones?name=%s", domain.DomainName),
		nil,
		&result,
	)

	return
}

// request 统一请求接口
func (hw *Huaweicloud) request(method string, url string, data interface{}, result interface{}) (err error) {
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
		hw.logger.Infof("http.NewRequest失败. Error: ", err)
		return
	}

	s := Signer{
		Key:    hw.DNS.ID,
		Secret: hw.DNS.Secret,
	}
	s.Sign(req)

	req.Header.Add("content-type", "application/json")

	client := util.CreateHTTPClient()
	resp, err := client.Do(req)
	err = util.GetHTTPResponse(resp, url, err, result)

	return
}
