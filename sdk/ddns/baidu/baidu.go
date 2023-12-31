package baidu

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

// https://cloud.baidu.com/doc/BCD/s/4jwvymhs7

const (
	Endpoint = "https://bcd.baidubce.com"
	Code     = "baidu"
)

type BaiduCloud struct {
	DNS     *config.DNS
	Domains ddns.Domains
	TTL     int
	logger  logger.ILogger
}

// BaiduRecord 单条解析记录
type BaiduRecord struct {
	RecordId uint   `json:"recordId"`
	Domain   string `json:"domain"`
	View     string `json:"view"`
	Rdtype   string `json:"rdtype"`
	TTL      int    `json:"ttl"`
	Rdata    string `json:"rdata"`
	ZoneName string `json:"zoneName"`
	Status   string `json:"status"`
}

// BaiduRecordsResp 获取解析列表拿到的结果
type BaiduRecordsResp struct {
	TotalCount int           `json:"totalCount"`
	Result     []BaiduRecord `json:"result"`
}

// BaiduListRequest 获取解析列表请求的body json
type BaiduListRequest struct {
	Domain   string `json:"domain"`
	PageNum  int    `json:"pageNum"`
	PageSize int    `json:"pageSize"`
}

// BaiduModifyRequest 修改解析请求的body json
type BaiduModifyRequest struct {
	RecordId uint   `json:"recordId"`
	Domain   string `json:"domain"`
	View     string `json:"view"`
	RdType   string `json:"rdType"`
	TTL      int    `json:"ttl"`
	Rdata    string `json:"rdata"`
	ZoneName string `json:"zoneName"`
}

// BaiduCreateRequest 创建新解析请求的body json
type BaiduCreateRequest struct {
	Domain   string `json:"domain"`
	RdType   string `json:"rdType"`
	TTL      int    `json:"ttl"`
	Rdata    string `json:"rdata"`
	ZoneName string `json:"zoneName"`
}

func (baidu *BaiduCloud) String() string {
	return Code
}

func (baidu *BaiduCloud) Endpoint() string {
	return Endpoint
}

func (baidu *BaiduCloud) Init(dnsConf *config.DDnsConfig, ipv4cache cache.IIpCache, ipv6cache cache.IIpCache, log logger.ILogger) {
	baidu.Domains.Ipv4Cache = ipv4cache
	baidu.Domains.Ipv6Cache = ipv6cache
	baidu.Domains.Logger = log
	baidu.DNS = dnsConf.DNS
	baidu.Domains.GetNewIp(dnsConf)
	baidu.logger = log
	if dnsConf.TTL == "" {
		// 默认300s
		baidu.TTL = 300
	} else {
		ttl, err := strconv.Atoi(dnsConf.TTL)
		if err != nil {
			baidu.TTL = 300
		} else {
			baidu.TTL = ttl
		}
	}
}

// AddUpdateDomainRecords 添加或更新IPv4/IPv6记录
func (baidu *BaiduCloud) AddUpdateDomainRecords() ddns.Domains {
	baidu.addUpdateDomainRecords("A")
	baidu.addUpdateDomainRecords("AAAA")
	return baidu.Domains
}

func (baidu *BaiduCloud) addUpdateDomainRecords(recordType string) {
	ipAddr, domains := baidu.Domains.GetNewIpResult(recordType)
	if ipAddr == "" {
		return
	}

	for _, domain := range domains {
		var records BaiduRecordsResp

		requestBody := BaiduListRequest{
			Domain:   domain.DomainName,
			PageNum:  1,
			PageSize: 1000,
		}

		err := baidu.request("POST", Endpoint+"/v1/domain/resolve/list", requestBody, &records)
		if err != nil {
			domain.UpdateStatus = consts.UpdatedFailed
			return
		}

		find := false
		for _, record := range records.Result {
			if record.Domain == domain.GetSubDomain() {
				//存在就去更新
				baidu.modify(record, domain, recordType, ipAddr)
				find = true
				break
			}
		}
		if !find {
			//没找到，去创建
			baidu.create(domain, recordType, ipAddr)
		}
	}
}

// create 创建新的解析
func (baidu *BaiduCloud) create(domain *ddns.Domain, recordType string, ipAddr string) {
	var baiduCreateRequest = BaiduCreateRequest{
		Domain:   domain.GetSubDomain(), //处理一下@
		RdType:   recordType,
		TTL:      baidu.TTL,
		Rdata:    ipAddr,
		ZoneName: domain.DomainName,
	}
	var result BaiduRecordsResp

	err := baidu.request("POST", Endpoint+"/v1/domain/resolve/add", baiduCreateRequest, &result)
	if err == nil {
		baidu.logger.Infof("新增域名解析 %s 成功！IP: %s", domain, ipAddr)
		domain.UpdateStatus = consts.UpdatedSuccess
	} else {
		baidu.logger.Infof("新增域名解析 %s 失败！", domain)
		domain.UpdateStatus = consts.UpdatedFailed
	}
}

// modify 更新解析
func (baidu *BaiduCloud) modify(record BaiduRecord, domain *ddns.Domain, rdType string, ipAddr string) {
	//没有变化直接跳过
	if record.Rdata == ipAddr {
		baidu.logger.Infof("你的IP %s 没有变化, 域名 %s", ipAddr, domain)
		return
	}
	var baiduModifyRequest = BaiduModifyRequest{
		RecordId: record.RecordId,
		Domain:   record.Domain,
		View:     record.View,
		RdType:   rdType,
		TTL:      record.TTL,
		Rdata:    ipAddr,
		ZoneName: record.ZoneName,
	}
	var result BaiduRecordsResp

	err := baidu.request("POST", Endpoint+"/v1/domain/resolve/edit", baiduModifyRequest, &result)
	if err == nil {
		baidu.logger.Infof("更新域名解析 %s 成功！IP: %s", domain, ipAddr)
		domain.UpdateStatus = consts.UpdatedSuccess
	} else {
		baidu.logger.Infof("更新域名解析 %s 失败！", domain)
		domain.UpdateStatus = consts.UpdatedFailed
	}
}

// request 统一请求接口
func (baidu *BaiduCloud) request(method string, url string, data interface{}, result interface{}) (err error) {
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
		baidu.logger.Infof("http.NewRequest失败. Error: ", err)
		return
	}

	BaiduSigner(baidu.DNS.ID, baidu.DNS.Secret, req)

	client := util.CreateHTTPClient()
	resp, err := client.Do(req)
	err = util.GetHTTPResponse(resp, url, err, result)

	return
}
