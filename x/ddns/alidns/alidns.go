package alidns

import (
	"bytes"
	"github.com/jxo-me/ddns/cache"
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/consts"
	"github.com/jxo-me/ddns/internal/util"
	"log"
	"net/http"
	"net/url"
)

const (
	Endpoint string = "https://alidns.aliyuncs.com/"
	Code     string = "alidns"
)

type Config struct {
	AccessKeyID     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
}

// Alidns Alidns
// https://help.aliyun.com/document_detail/29776.html?spm=a2c4g.11186623.6.672.715a45caji9dMA
type Alidns struct {
	DNS     *config.DNS
	Domains config.Domains
	TTL     string
}

// AlidnsRecord record
type AlidnsRecord struct {
	DomainName string
	RecordID   string
	Value      string
}

// AlidnsSubDomainRecords 记录
type AlidnsSubDomainRecords struct {
	TotalCount    int
	DomainRecords struct {
		Record []AlidnsRecord
	}
}

// AlidnsResp 修改/添加返回结果
type AlidnsResp struct {
	RecordID  string
	RequestID string
}

func (ali *Alidns) String() string {
	return Code
}

func (ali *Alidns) Endpoint() string {
	return Endpoint
}

// Init 初始化
func (ali *Alidns) Init(dnsConf *config.DnsConfig, ipv4cache *cache.IpCache, ipv6cache *cache.IpCache) {
	ali.Domains.Ipv4Cache = ipv4cache
	ali.Domains.Ipv6Cache = ipv6cache
	ali.DNS = dnsConf.DNS
	ali.Domains.GetNewIp(dnsConf)
	if dnsConf.TTL == "" {
		// 默认600s
		ali.TTL = "600"
	} else {
		ali.TTL = dnsConf.TTL
	}
}

// AddUpdateDomainRecords 添加或更新IPv4/IPv6记录
func (ali *Alidns) AddUpdateDomainRecords() config.Domains {
	ali.addUpdateDomainRecords("A")
	ali.addUpdateDomainRecords("AAAA")
	return ali.Domains
}

func (ali *Alidns) addUpdateDomainRecords(recordType string) {
	ipAddr, domains := ali.Domains.GetNewIpResult(recordType)

	if ipAddr == "" {
		return
	}

	for _, domain := range domains {
		var records AlidnsSubDomainRecords
		// 获取当前域名信息
		params := domain.GetCustomParams()
		params.Set("Action", "DescribeSubDomainRecords")
		params.Set("DomainName", domain.DomainName)
		params.Set("SubDomain", domain.GetFullDomain())
		params.Set("Type", recordType)
		err := ali.request(params, &records)

		if err != nil {
			domain.UpdateStatus = consts.UpdatedFailed
			return
		}

		if records.TotalCount > 0 {
			// 默认第一个
			recordSelected := records.DomainRecords.Record[0]
			if params.Has("RecordId") {
				for i := 0; i < len(records.DomainRecords.Record); i++ {
					if records.DomainRecords.Record[i].RecordID == params.Get("RecordId") {
						recordSelected = records.DomainRecords.Record[i]
					}
				}
			}
			// 存在，更新
			ali.modify(recordSelected, domain, recordType, ipAddr)
		} else {
			// 不存在，创建
			ali.create(domain, recordType, ipAddr)
		}

	}
}

// 创建
func (ali *Alidns) create(domain *config.Domain, recordType string, ipAddr string) {
	params := domain.GetCustomParams()
	params.Set("Action", "AddDomainRecord")
	params.Set("DomainName", domain.DomainName)
	params.Set("RR", domain.GetSubDomain())
	params.Set("Type", recordType)
	params.Set("Value", ipAddr)
	params.Set("TTL", ali.TTL)

	var result AlidnsResp
	err := ali.request(params, &result)

	if err == nil && result.RecordID != "" {
		log.Printf("新增域名解析 %s 成功！IP: %s", domain, ipAddr)
		domain.UpdateStatus = consts.UpdatedSuccess
	} else {
		log.Printf("新增域名解析 %s 失败！", domain)
		domain.UpdateStatus = consts.UpdatedFailed
	}
}

// 修改
func (ali *Alidns) modify(recordSelected AlidnsRecord, domain *config.Domain, recordType string, ipAddr string) {

	// 相同不修改
	if recordSelected.Value == ipAddr {
		log.Printf("你的IP %s 没有变化, 域名 %s", ipAddr, domain)
		return
	}

	params := domain.GetCustomParams()
	params.Set("Action", "UpdateDomainRecord")
	params.Set("RR", domain.GetSubDomain())
	params.Set("RecordId", recordSelected.RecordID)
	params.Set("Type", recordType)
	params.Set("Value", ipAddr)
	params.Set("TTL", ali.TTL)

	var result AlidnsResp
	err := ali.request(params, &result)

	if err == nil && result.RecordID != "" {
		log.Printf("更新域名解析 %s 成功！IP: %s", domain, ipAddr)
		domain.UpdateStatus = consts.UpdatedSuccess
	} else {
		log.Printf("更新域名解析 %s 失败！", domain)
		domain.UpdateStatus = consts.UpdatedFailed
	}
}

// request 统一请求接口
func (ali *Alidns) request(params url.Values, result interface{}) (err error) {

	AliyunSigner(ali.DNS.ID, ali.DNS.Secret, &params)

	req, err := http.NewRequest(
		"GET",
		Endpoint,
		bytes.NewBuffer(nil),
	)
	req.URL.RawQuery = params.Encode()

	if err != nil {
		log.Println("http.NewRequest失败. Error: ", err)
		return
	}

	client := util.CreateHTTPClient()
	resp, err := client.Do(req)
	err = util.GetHTTPResponse(resp, Endpoint, err, result)

	return
}
