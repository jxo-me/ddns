package tencent

import (
	"bytes"
	"encoding/json"
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
	Endpoint            = "https://dnspod.tencentcloudapi.com"
	tencentCloudVersion = "2021-03-23"
	Code                = "tencent"
)

// TencentCloud 腾讯云 DNSPod API 3.0 实现
// https://cloud.tencent.com/document/api/1427/56193
type TencentCloud struct {
	DNS     *config.DNS
	Domains ddns.Domains
	TTL     int
	logger  logger.ILogger
}

// TencentCloudRecord 腾讯云记录
type TencentCloudRecord struct {
	Domain string `json:"Domain"`
	// DescribeRecordList 不需要 SubDomain
	SubDomain string `json:"SubDomain,omitempty"`
	// CreateRecord/ModifyRecord 不需要 Subdomain
	Subdomain  string `json:"Subdomain,omitempty"`
	RecordType string `json:"RecordType"`
	RecordLine string `json:"RecordLine"`
	// DescribeRecordList 不需要 Value
	Value string `json:"Value,omitempty"`
	// CreateRecord/DescribeRecordList 不需要 RecordId
	RecordId int `json:"RecordId,omitempty"`
	// DescribeRecordList 不需要 TTL
	TTL int `json:"TTL,omitempty"`
}

// TencentCloudRecordListsResp 获取域名的解析记录列表返回结果
type TencentCloudRecordListsResp struct {
	TencentCloudStatus
	Response struct {
		RecordCountInfo struct {
			TotalCount int `json:"TotalCount"`
		} `json:"RecordCountInfo"`

		RecordList []TencentCloudRecord `json:"RecordList"`
	}
}

// TencentCloudStatus 腾讯云返回状态
// https://cloud.tencent.com/document/product/1427/56192
type TencentCloudStatus struct {
	Response struct {
		Error struct {
			Code    string
			Message string
		}
	}
}

func (tc *TencentCloud) String() string {
	return Code
}

func (tc *TencentCloud) Endpoint() string {
	return Endpoint
}

func (tc *TencentCloud) Init(dnsConf *config.DDnsConfig, ipv4cache cache.IIpCache, ipv6cache cache.IIpCache, log logger.ILogger) {
	tc.Domains.Ipv4Cache = ipv4cache
	tc.Domains.Ipv6Cache = ipv6cache
	tc.Domains.Logger = log
	tc.DNS = dnsConf.DNS
	tc.Domains.GetNewIp(dnsConf)
	tc.logger = log
	if dnsConf.TTL == "" {
		// 默认 600s
		tc.TTL = 600
	} else {
		ttl, err := strconv.Atoi(dnsConf.TTL)
		if err != nil {
			tc.TTL = 600
		} else {
			tc.TTL = ttl
		}
	}
}

// AddUpdateDomainRecords 添加或更新 IPv4/IPv6 记录
func (tc *TencentCloud) AddUpdateDomainRecords() ddns.Domains {
	tc.addUpdateDomainRecords("A")
	tc.addUpdateDomainRecords("AAAA")
	return tc.Domains
}

func (tc *TencentCloud) addUpdateDomainRecords(recordType string) {
	ipAddr, domains := tc.Domains.GetNewIpResult(recordType)

	if ipAddr == "" {
		return
	}

	for _, domain := range domains {
		result, err := tc.getRecordList(domain, recordType)
		if err != nil {
			domain.UpdateStatus = consts.UpdatedFailed
			return
		}

		if result.Response.RecordCountInfo.TotalCount > 0 {
			// 默认第一个
			recordSelected := result.Response.RecordList[0]
			params := domain.GetCustomParams()
			if params.Has("RecordId") {
				for i := 0; i < result.Response.RecordCountInfo.TotalCount; i++ {
					if strconv.Itoa(result.Response.RecordList[i].RecordId) == params.Get("RecordId") {
						recordSelected = result.Response.RecordList[i]
					}
				}
			}

			// 修改记录
			tc.modify(recordSelected, domain, recordType, ipAddr)
		} else {
			// 添加记录
			tc.create(domain, recordType, ipAddr)
		}
	}
}

// create 添加记录
// CreateRecord https://cloud.tencent.com/document/api/1427/56180
func (tc *TencentCloud) create(domain *ddns.Domain, recordType string, ipAddr string) {
	record := &TencentCloudRecord{
		Domain:     domain.DomainName,
		SubDomain:  domain.GetSubDomain(),
		RecordType: recordType,
		RecordLine: tc.getRecordLine(domain),
		Value:      ipAddr,
		TTL:        tc.TTL,
	}

	var status TencentCloudStatus
	err := tc.request(
		"CreateRecord",
		record,
		&status,
	)
	if err == nil && status.Response.Error.Code == "" {
		tc.logger.Infof("新增域名解析 %s 成功！IP: %s", domain, ipAddr)
		domain.UpdateStatus = consts.UpdatedSuccess
	} else {
		tc.logger.Infof("新增域名解析 %s 失败！Code: %s, Message: %s", domain, status.Response.Error.Code, status.Response.Error.Message)
		domain.UpdateStatus = consts.UpdatedFailed
	}
}

// modify 修改记录
// ModifyRecord https://cloud.tencent.com/document/api/1427/56157
func (tc *TencentCloud) modify(record TencentCloudRecord, domain *ddns.Domain, recordType string, ipAddr string) {
	// 相同不修改
	if record.Value == ipAddr {
		tc.logger.Infof("你的IP %s 没有变化, 域名 %s", ipAddr, domain)
		return
	}
	var status TencentCloudStatus
	record.Domain = domain.DomainName
	record.SubDomain = domain.GetSubDomain()
	record.RecordType = recordType
	record.RecordLine = tc.getRecordLine(domain)
	record.Value = ipAddr
	record.TTL = tc.TTL
	err := tc.request(
		"ModifyRecord",
		record,
		&status,
	)
	if err == nil && status.Response.Error.Code == "" {
		tc.logger.Infof("更新域名解析 %s 成功！IP: %s", domain, ipAddr)
		domain.UpdateStatus = consts.UpdatedSuccess
	} else {
		tc.logger.Infof("更新域名解析 %s 失败！Code: %s, Message: %s", domain, status.Response.Error.Code, status.Response.Error.Message)
		domain.UpdateStatus = consts.UpdatedFailed
	}
}

// getRecordList 获取域名的解析记录列表
// DescribeRecordList https://cloud.tencent.com/document/api/1427/56166
func (tc *TencentCloud) getRecordList(domain *ddns.Domain, recordType string) (result TencentCloudRecordListsResp, err error) {
	record := TencentCloudRecord{
		Domain:     domain.DomainName,
		Subdomain:  domain.GetSubDomain(),
		RecordType: recordType,
		RecordLine: tc.getRecordLine(domain),
	}
	err = tc.request(
		"DescribeRecordList",
		record,
		&result,
	)

	return
}

// getRecordLine 获取记录线路，为空返回默认
func (tc *TencentCloud) getRecordLine(domain *ddns.Domain) string {
	if domain.GetCustomParams().Has("RecordLine") {
		return domain.GetCustomParams().Get("RecordLine")
	}
	return "默认"
}

// request 统一请求接口
func (tc *TencentCloud) request(action string, data interface{}, result interface{}) (err error) {
	jsonStr := make([]byte, 0)
	if data != nil {
		jsonStr, _ = json.Marshal(data)
	}
	req, err := http.NewRequest(
		"POST",
		Endpoint,
		bytes.NewBuffer(jsonStr),
	)
	if err != nil {
		tc.logger.Infof("http.NewRequest 失败. Error: ", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TC-Version", tencentCloudVersion)

	TencentCloudSigner(tc.DNS.ID, tc.DNS.Secret, req, action, string(jsonStr))

	client := util.CreateHTTPClient()
	resp, err := client.Do(req)
	err = util.GetHTTPResponse(resp, Endpoint, err, result)

	return
}
