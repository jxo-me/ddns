package hook

import (
	"encoding/json"
	"fmt"
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/consts"
	"github.com/jxo-me/ddns/internal/util"
	"log"
	"net/http"
	"net/url"
	"strings"
)

const (
	Code = "webhook"
)

// Webhook Webhook
type Webhook struct {
	WebhookURL         string
	WebhookRequestBody string
	WebhookHeaders     string
}

// hasJSONPrefix returns true if the string starts with a JSON open brace.
func hasJSONPrefix(s string) bool {
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}

func NewHook(url string, requestBody string, headers string) *Webhook {
	return &Webhook{
		WebhookURL:         url,
		WebhookRequestBody: requestBody,
		WebhookHeaders:     headers,
	}
}

func (w *Webhook) String() string {
	return Code
}

// ExecHook 添加或更新IPv4/IPv6记录, 返回是否有更新失败的
func (w *Webhook) ExecHook(domains *config.Domains) (v4Status consts.UpdateStatusType, v6Status consts.UpdateStatusType) {
	v4Status = w.getDomainsStatus(domains.Ipv4Domains)
	v6Status = w.getDomainsStatus(domains.Ipv6Domains)

	if w.WebhookURL != "" && (v4Status != consts.UpdatedNothing || v6Status != consts.UpdatedNothing) {
		// 成功和失败都要触发webhook
		method := "GET"
		postPara := ""
		contentType := "application/x-www-form-urlencoded"
		if w.WebhookRequestBody != "" {
			method = "POST"
			postPara = w.replacePara(domains, w.WebhookRequestBody, v4Status, v6Status)
			if json.Valid([]byte(postPara)) {
				contentType = "application/json"
				// 如果 RequestBody 的 JSON 无效但前缀为 JSON 括号则为 JSON
			} else if hasJSONPrefix(postPara) {
				log.Println("RequestBody 的 JSON 无效！")
			}
		}
		requestURL := w.replacePara(domains, w.WebhookURL, v4Status, v6Status)
		u, err := url.Parse(requestURL)
		if err != nil {
			log.Println("Webhook配置中的URL不正确")
			return
		}
		req, err := http.NewRequest(method, fmt.Sprintf("%s://%s%s?%s", u.Scheme, u.Host, u.Path, u.Query().Encode()), strings.NewReader(postPara))
		if err != nil {
			log.Println("创建Webhook请求异常, Err:", err)
			return
		}

		headers := w.CheckParseHeaders(w.WebhookHeaders)
		for key, value := range headers {
			req.Header.Add(key, value)
		}
		req.Header.Add("content-type", contentType)

		clt := util.CreateHTTPClient()
		resp, err := clt.Do(req)
		body, err := util.GetHTTPResponseOrg(resp, requestURL, err)
		if err == nil {
			log.Printf("Webhook调用成功, 返回数据: %q\n", string(body))
		} else {
			log.Printf("Webhook调用失败，Err：%s\n", err)
		}
	}
	return
}

// getDomainsStatus 获取域名状态
func (w *Webhook) getDomainsStatus(domains []*config.Domain) consts.UpdateStatusType {
	successNum := 0
	for _, v46 := range domains {
		switch v46.UpdateStatus {
		case consts.UpdatedFailed:
			// 一个失败，全部失败
			return consts.UpdatedFailed
		case consts.UpdatedSuccess:
			successNum++
		}
	}

	if successNum > 0 {
		// 迭代完成后一个成功，就成功
		return consts.UpdatedSuccess
	}
	return consts.UpdatedNothing
}

// replacePara 替换参数
func (w *Webhook) replacePara(domains *config.Domains, orgPara string, ipv4Result consts.UpdateStatusType, ipv6Result consts.UpdateStatusType) (newPara string) {
	orgPara = strings.ReplaceAll(orgPara, "#{ipv4Addr}", domains.Ipv4Addr)
	orgPara = strings.ReplaceAll(orgPara, "#{ipv4Result}", string(ipv4Result))
	orgPara = strings.ReplaceAll(orgPara, "#{ipv4Domains}", w.getDomainsStr(domains.Ipv4Domains))

	orgPara = strings.ReplaceAll(orgPara, "#{ipv6Addr}", domains.Ipv6Addr)
	orgPara = strings.ReplaceAll(orgPara, "#{ipv6Result}", string(ipv6Result))
	orgPara = strings.ReplaceAll(orgPara, "#{ipv6Domains}", w.getDomainsStr(domains.Ipv6Domains))

	return orgPara
}

// getDomainsStr 用逗号分割域名
func (w *Webhook) getDomainsStr(domains []*config.Domain) string {
	str := ""
	for i, v46 := range domains {
		str += v46.String()
		if i != len(domains)-1 {
			str += ","
		}
	}

	return str
}

func (w *Webhook) CheckParseHeaders(headerStr string) (headers map[string]string) {
	headers = make(map[string]string)
	headerArr := strings.Split(headerStr, "\r\n")
	for _, headerStr := range headerArr {
		headerStr = strings.TrimSpace(headerStr)
		if headerStr != "" {
			parts := strings.Split(headerStr, ":")
			if len(parts) != 2 {
				log.Println(headerStr, "Header不正确")
				continue
			}
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return headers
}
