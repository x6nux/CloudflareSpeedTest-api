package cfip

import (
	"crypto/tls"
	_const "edulx/CloudflareSpeedTest-api/const"
	"edulx/CloudflareSpeedTest-api/task"
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type CloudflareAPI struct {
	Clock       int  `yaml:"clock"`
	ClockSwitch bool `yaml:"clock_switch"`
	TGbot       struct {
		TGbotToken  string `yaml:"tgbot_token"`
		TGbotChatID string `yaml:"tgbot_chat_id"`
		Switch      bool   `yaml:"switch"`
	} `yaml:"tgbot"`
	Email     string   `yaml:"email"`
	ApiKeys   string   `yaml:"api_key"`
	ZoneId    string   `yaml:"zone_id"`
	Domain    string   `yaml:"domain"`
	SubDomain []string `yaml:"subdomains"`
}

var C CloudflareAPI

// ReadYaml 读取yaml文件
func (c *CloudflareAPI) ReadYaml() error {
	//读取yaml文件
	yamlFile, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(yamlFile, &C)
	if err != nil {
		return err
	}
	if C.Email == "" || C.ApiKeys == "" || C.ZoneId == "" || C.Domain == "" || len(C.SubDomain) == 0 {
		fmt.Println("请检查config.yaml配置文件是否正确")
		_const.TGPUSH += "请检查config.yaml配置文件是否正确\n"
		return errors.New("请检查config.yaml配置文件是否正确")
	}
	return nil
}

// GetDomain 调用cloudflare的api查询对应的域名
func (c *CloudflareAPI) GetDomain(ip net.IPAddr) (CFR, error) {
	url := "https://api.cloudflare.com/client/v4/zones/" + c.ZoneId + "/dns_records?page=1&per_page=20&order=type&direction=asc"
	client := &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			DialContext:     task.GetDialContext(&ip),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // 跳过证书验证
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 阻止重定向
		},
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return CFR{}, err
	}
	req.Header.Add("X-Auth-Email", c.Email)
	req.Header.Add("X-Auth-Key", c.ApiKeys)
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return CFR{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
		
		}
	}(res.Body)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return CFR{}, err
	}
	var s CFR
	err = json.Unmarshal(body, &s)
	if err != nil {
		return CFR{}, err
	}
	if s.Success == false {
		return CFR{}, errors.New("域名信息获取失败")
	}
	return s, nil
}

// GetDomainUuid 筛选出对应的域名
func (c *CloudflareAPI) GetDomainUuid(ip net.IPAddr) (map[string]string, error) {
	s, err := c.GetDomain(ip)
	if err != nil {
		return nil, err
	}
	if s.Success == false || s.Errors == nil {
		return nil, err
	}
	mp := make(map[string]string)
	for _, v := range s.Result {
		mp[strings.Split(v.Name, ".")[0]] = v.Id
	}
	return mp, nil
}

// SortIp 根据速度对ip进行排序从大到小
func (c *CloudflareAPI) SortIp(ip []net.IPAddr, speed []float64) []net.IPAddr {
	for i := 0; i < len(speed); i++ {
		for j := i + 1; j < len(speed); j++ {
			if speed[i] < speed[j] {
				speed[i], speed[j] = speed[j], speed[i]
				ip[i], ip[j] = ip[j], ip[i]
			}
		}
	}
	return ip
}

// UpdateDomain 更新域名
func (c *CloudflareAPI) UpdateDomain(ip []net.IPAddr, speed []float64) {
	if len(ip) < len(c.SubDomain) {
		fmt.Println("ip地址和域名数量不匹配")
		_const.TGPUSH += "ip地址和域名数量不匹配\n"
		return
	}
	err := c.ReadYaml()
	if err != nil {
		return
	}
	var s map[string]string
	for i := 0; i <= len(ip); i++ {
		if i == len(ip) {
			fmt.Println("域名信息获取失败,结束重试")
			_const.TGPUSH += "域名信息获取失败,结束重试\n"
			return
		}
		s, err = c.GetDomainUuid(ip[i])
		if err == nil {
			break
		}
		fmt.Println("域名信息第" + strconv.Itoa(i+1) + "次获取失败,正在进行下一次重试")
	}
	
	ip = c.SortIp(ip, speed)
	for i, v := range c.SubDomain {
		if i >= len(ip) {
			break
		}
		if s[v] == "" {
			fmt.Println("域名:" + v + "." + c.Domain + "不存在")
			for k := 0; k <= len(ip); k++ {
				if k == len(ip) {
					fmt.Println("域名" + v + "." + c.Domain + "创建失败 IP:" + ip[k].String())
					_const.TGPUSH += "域名" + v + "." + c.Domain + "创建失败 IP:" + ip[k].String() + "\n"
					break
				}
				err := c.CreateDomain(v, ip[i], ip[k])
				if err == nil {
					fmt.Println("域名" + v + "." + c.Domain + "创建成功 IP:" + ip[k].String())
					_const.TGPUSH += "域名" + v + "." + c.Domain + "创建成功 IP:" + ip[k].String() + "\n"
					break
				}
				fmt.Println("域名" + v + "." + c.Domain + "第" + strconv.Itoa(k+1) + "次创建失败,正在进行下一次重试")
			}
			
			continue
		}
		for t := 0; t <= len(ip); t++ {
			if t == len(ip) {
				fmt.Println("域名" + v + "." + c.Domain + "更新失败 IP:" + ip[i].String())
				_const.TGPUSH += "域名" + v + "." + c.Domain + "更新失败 IP:" + ip[i].String() + "\n"
				break
			}
			err := c.PUTDomains(ip[i], v, ip[t], s[v])
			if err == nil {
				fmt.Println("域名" + v + "." + c.Domain + "更新成功 IP:" + ip[i].String())
				_const.TGPUSH += "域名" + v + "." + c.Domain + "更新成功 IP:" + ip[i].String() + "\n"
				break
			}
			fmt.Println("域名" + v + "." + c.Domain + "第" + strconv.Itoa(t+1) + "次更新失败,正在进行下一次重试")
		}
	}
}
func (c *CloudflareAPI) PUTDomains(ip net.IPAddr, subdomain string, ips net.IPAddr, domainid string) error {
	url := "https://api.cloudflare.com/client/v4/zones/" + c.ZoneId + "/dns_records/" + domainid
	method := "PUT"
	payload := strings.NewReader(`{"type":"A","name":"` + subdomain + `.` + c.Domain + `","content":"` + ip.String() + `","ttl":60,"proxied":false}`)
	client := &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			DialContext:     task.GetDialContext(&ips),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // 跳过证书验证
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 阻止重定向
		},
	}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return err
	}
	req.Header.Add("X-Auth-Email", c.Email)
	req.Header.Add("X-Auth-Key", c.ApiKeys)
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(res.Body)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err.Error())
		}
	}(res.Body)
	if err != nil {
		return err
	}
	var s map[string]interface{}
	err = json.Unmarshal(body, &s)
	if err != nil {
		return err
	}
	if s["success"] == false {
		
		return errors.New("更新失败")
	} else {
		return nil
	}
}

// CreateDomain 创建域名解析
func (c *CloudflareAPI) CreateDomain(subdomain string, ip net.IPAddr, ips net.IPAddr) error {
	err := c.ReadYaml()
	if err != nil {
		return err
	}
	url := "https://api.cloudflare.com/client/v4/zones/" + c.ZoneId + "/dns_records"
	method := "POST"
	payload := strings.NewReader(`{"type":"A","name":"` + subdomain + `.` + c.Domain + `","content":"` + ip.String() + `","ttl":60,"proxied":false}`)
	client := &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			DialContext:     task.GetDialContext(&ips),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // 跳过证书验证
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 阻止重定向
		}}
	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return err
	}
	req.Header.Add("X-Auth-Email", c.Email)
	req.Header.Add("X-Auth-Key", c.ApiKeys)
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	body, err := io.ReadAll(res.Body)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err.Error())
		}
	}(res.Body)
	if err != nil {
		return err
	}
	var s map[string]interface{}
	err = json.Unmarshal(body, &s)
	if err != nil {
		return err
	}
	if s["success"] == false {
		return errors.New("创建失败")
	} else {
		return nil
	}
	
}
