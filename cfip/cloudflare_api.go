package cfip

import (
	"encoding/json"
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type CloudflareAPI struct {
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
		return errors.New("请检查config.yaml配置文件是否正确")
	}
	return nil
}

// GetDomain 调用cloudflare的api查询对应的域名
func (c *CloudflareAPI) GetDomain() CFR {
	url := "https://api.cloudflare.com/client/v4/zones/" + c.ZoneId + "/dns_records?page=1&per_page=20&order=type&direction=asc"
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err.Error())
		return CFR{}
	}
	req.Header.Add("X-Auth-Email", c.Email)
	req.Header.Add("X-Auth-Key", c.ApiKeys)
	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return CFR{}
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(res.Body)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err.Error())
		return CFR{}
	}
	var s CFR
	err = json.Unmarshal(body, &s)
	if err != nil {
		fmt.Println(err.Error())
		return CFR{}
	}
	return s
}

// GetDomainUuid 筛选出对应的域名
func (c *CloudflareAPI) GetDomainUuid() map[string]string {
	s := c.GetDomain()
	mp := make(map[string]string)
	for _, v := range s.Result {
		mp[strings.Split(v.Name, ".")[0]] = v.Id
	}
	return mp
}

// SortIp 根据速度对ip进行排序从大到小
func (c *CloudflareAPI) SortIp(ip []string, speed []float64) []string {
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
func (c *CloudflareAPI) UpdateDomain(ip []string, speed []float64) {
	if len(ip) < len(c.SubDomain) {
		fmt.Println("ip地址和域名数量不匹配")
		return
	}
	err := c.ReadYaml()
	if err != nil {
		return
	}
	s := c.GetDomainUuid()
	ip = c.SortIp(ip, speed)
	for i, v := range c.SubDomain {
		if s[v] == "" {
			fmt.Println("域名:" + v + "." + c.Domain + "不存在")
			continue
		}
		url := "https://api.cloudflare.com/client/v4/zones/" + c.ZoneId + "/dns_records/" + s[v]
		method := "PUT"
		payload := strings.NewReader(`{"type":"A","name":"` + v + `.` + c.Domain + `","content":"` + ip[i] + `","ttl":60,"proxied":false}`)
		client := &http.Client{}
		req, err := http.NewRequest(method, url, payload)
		if err != nil {
			fmt.Println(err.Error())
		}
		req.Header.Add("X-Auth-Email", c.Email)
		req.Header.Add("X-Auth-Key", c.ApiKeys)
		req.Header.Add("Content-Type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			fmt.Println(err.Error())
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Println(err.Error())
		}
		var s map[string]interface{}
		err = json.Unmarshal(body, &s)
		if err != nil {
			fmt.Println(err.Error())
		}
		if s["success"] == false {
			fmt.Println("域名:" + v + "." + c.Domain + "更新失败 IP:" + ip[i])
		} else {
			fmt.Println("域名:" + v + "." + c.Domain + "更新成功 IP:" + ip[i])
		}
		err = res.Body.Close()
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}
}
