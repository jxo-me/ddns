package config

import (
	"github.com/jxo-me/ddns/internal/util"
	"io"
	"log"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Ipv4Reg IPv4正则
var Ipv4Reg = regexp.MustCompile(`((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])`)

// Ipv6Reg IPv6正则
var Ipv6Reg = regexp.MustCompile(`((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))`)

// DNS DNS配置
type DNS struct {
	// 名称。如：alidns,webhook
	Name   string `json:"name"`
	ID     string `json:"ID"`
	Secret string `json:"secret"`
}

type Ipv4 struct {
	Enable bool `json:"enable"`
	// 获取IP类型 url/netInterface
	GetType      string   `json:"getType"`
	URL          string   `json:"url"`
	NetInterface string   `json:"netInterface"`
	Cmd          string   `json:"cmd"`
	Domains      []string `json:"domains"`
}

type Ipv6 struct {
	Enable bool `json:"enable"`
	// 获取IP类型 url/netInterface
	GetType      string `yaml:",omitempty" json:"getType"`
	URL          string `yaml:",omitempty" json:"url"`
	NetInterface string `yaml:",omitempty" json:"netInterface"`
	Cmd          string `yaml:",omitempty" json:"cmd"`
	// ipv6匹配正则表达式
	IPv6Reg string   `yaml:",omitempty" json:"IPv6Reg"`
	Domains []string `yaml:",omitempty" json:"domains"`
}

// DnsConfig 配置
type DnsConfig struct {
	Name    string   `json:"name"`
	Ipv4    *Ipv4    `yaml:",omitempty"  json:"ipv4"`
	Ipv6    *Ipv6    `yaml:",omitempty" json:"ipv6"`
	DNS     *DNS     `yaml:",omitempty" json:"dns"`
	TTL     string   `yaml:",omitempty" json:"ttl"`
	Webhook *Webhook `yaml:",omitempty" json:"webhook"`
}

func (conf *DnsConfig) getIpv4AddrFromInterface() string {
	ipv4, _, err := util.GetNetInterface()
	if err != nil {
		log.Println("从网卡获得IPv4失败!")
		return ""
	}

	for _, netInterface := range ipv4 {
		if netInterface.Name == conf.Ipv4.NetInterface && len(netInterface.Address) > 0 {
			return netInterface.Address[0]
		}
	}

	log.Println("从网卡中获得IPv4失败! 网卡名: ", conf.Ipv4.NetInterface)
	return ""
}

func (conf *DnsConfig) getIpv4AddrFromUrl() string {
	client := util.CreateNoProxyHTTPClient("tcp4")
	urls := strings.Split(conf.Ipv4.URL, ",")
	for _, url := range urls {
		url = strings.TrimSpace(url)
		resp, err := client.Get(url)
		if err != nil {
			log.Printf("连接失败! <a target='blank' href='%s'>点击查看接口能否返回IPv4地址</a>\n", url)
			log.Printf("错误信息: %s\n", err)
			continue
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("读取IPv4结果失败! 接口: ", url)
			continue
		}
		result := Ipv4Reg.FindString(string(body))
		if result == "" {
			log.Printf("获取IPv4结果失败! 接口: %s ,返回值: %s\n", url, result)
		}
		return result
	}
	return ""
}

func (conf *DnsConfig) getAddrFromCmd(addrType string) string {
	var cmd string
	var comp *regexp.Regexp
	if addrType == "IPv4" {
		cmd = conf.Ipv4.Cmd
		comp = Ipv4Reg
	} else {
		cmd = conf.Ipv6.Cmd
		comp = Ipv6Reg
	}
	// cmd is empty
	if cmd == "" {
		return ""
	}
	// run cmd with proper shell
	var execCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		execCmd = exec.Command("powershell", "-Command", cmd)
	} else {
		// If Bash does not exist, use sh
		_, err := exec.LookPath("bash")
		if err != nil {
			execCmd = exec.Command("sh", "-c", cmd)
		} else {
			execCmd = exec.Command("bash", "-c", cmd)
		}
	}
	// run cmd
	out, err := execCmd.CombinedOutput()
	if err != nil {
		log.Printf("获取%s结果失败! 未能成功执行命令：%s，错误：%q，退出状态码：%s\n", addrType, execCmd.String(), out, err)
		return ""
	}
	str := string(out)
	// get result
	result := comp.FindString(str)
	if result == "" {
		log.Printf("获取%s结果失败! 命令：%s，标准输出：%q\n", addrType, execCmd.String(), str)
	}
	return result
}

// GetIpv4Addr 获得IPv4地址
func (conf *DnsConfig) GetIpv4Addr() string {
	// 判断从哪里获取IP
	switch conf.Ipv4.GetType {
	case "netInterface":
		// 从网卡获取 IP
		return conf.getIpv4AddrFromInterface()
	case "url":
		// 从 URL 获取 IP
		return conf.getIpv4AddrFromUrl()
	case "cmd":
		// 从命令行获取 IP
		return conf.getAddrFromCmd("IPv4")
	default:
		log.Println("IPv4 的 获取 IP 方式 未知！")
		return "" // unknown type
	}
}

func (conf *DnsConfig) getIpv6AddrFromInterface() string {
	_, ipv6, err := util.GetNetInterface()
	if err != nil {
		log.Println("从网卡获得IPv6失败!")
		return ""
	}

	for _, netInterface := range ipv6 {
		if netInterface.Name == conf.Ipv6.NetInterface && len(netInterface.Address) > 0 {
			if conf.Ipv6.IPv6Reg != "" {
				// 匹配第几个IPv6
				if match, err := regexp.MatchString("@\\d", conf.Ipv6.IPv6Reg); err == nil && match {
					num, err := strconv.Atoi(conf.Ipv6.IPv6Reg[1:])
					if err == nil {
						if num > 0 {
							log.Printf("IPv6将使用第 %d 个IPv6地址\n", num)
							if num <= len(netInterface.Address) {
								return netInterface.Address[num-1]
							}
							log.Printf("未找到第 %d 个IPv6地址! 将使用第一个IPv6地址\n", num)
							return netInterface.Address[0]
						}
						log.Printf("IPv6匹配表达式 %s 不正确! 最小从1开始\n", conf.Ipv6.IPv6Reg)
						return ""
					}
				}
				// 正则表达式匹配
				log.Printf("IPv6将使用正则表达式 %s 进行匹配\n", conf.Ipv6.IPv6Reg)
				for i := 0; i < len(netInterface.Address); i++ {
					matched, err := regexp.MatchString(conf.Ipv6.IPv6Reg, netInterface.Address[i])
					if matched && err == nil {
						log.Println("匹配成功! 匹配到地址: ", netInterface.Address[i])
						return netInterface.Address[i]
					}
					log.Printf("第 %d 个地址 %s 不匹配, 将匹配下一个地址\n", i+1, netInterface.Address[i])
				}
				log.Println("没有匹配到任何一个IPv6地址, 将使用第一个地址")
			}
			return netInterface.Address[0]
		}
	}

	log.Println("从网卡中获得IPv6失败! 网卡名: ", conf.Ipv6.NetInterface)
	return ""
}

func (conf *DnsConfig) getIpv6AddrFromUrl() string {
	client := util.CreateNoProxyHTTPClient("tcp6")
	urls := strings.Split(conf.Ipv6.URL, ",")
	for _, url := range urls {
		url = strings.TrimSpace(url)
		resp, err := client.Get(url)
		if err != nil {
			log.Printf("连接失败! <a target='blank' href='%s'>点击查看接口能否返回IPv6地址</a>, 参考说明:<a target='blank' href='%s'>点击访问</a>\n", url, "https://github.com/jeessy2/ddns-go#使用ipv6")
			log.Printf("错误信息: %s\n", err)
			continue
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Println("读取IPv6结果失败! 接口: ", url)
			continue
		}
		result := Ipv6Reg.FindString(string(body))
		if result == "" {
			log.Printf("获取IPv6结果失败! 接口: %s ,返回值: %s\n", url, result)
		}
		return result
	}
	return ""
}

// GetIpv6Addr 获得IPv6地址
func (conf *DnsConfig) GetIpv6Addr() (result string) {
	// 判断从哪里获取IP
	switch conf.Ipv6.GetType {
	case "netInterface":
		// 从网卡获取 IP
		return conf.getIpv6AddrFromInterface()
	case "url":
		// 从 URL 获取 IP
		return conf.getIpv6AddrFromUrl()
	case "cmd":
		// 从命令行获取 IP
		return conf.getAddrFromCmd("IPv6")
	default:
		log.Println("IPv6 的 获取 IP 方式 未知！")
		return "" // unknown type
	}
}