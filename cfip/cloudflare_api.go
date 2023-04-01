package cfip

import (
	"encoding/json"
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
func (c *CloudflareAPI) ReadYaml() {
	//读取yaml文件
	yamlFile, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		fmt.Println(err.Error())
	}
	err = yaml.Unmarshal(yamlFile, &C)
	if err != nil {
		fmt.Println(err.Error())
	}
}

// 调用cloudfare的api查询对应的域名
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

// 筛选出对应的域名
func (c *CloudflareAPI) GetDomainUuid() map[string]string {
	s := c.GetDomain()
	mp := make(map[string]string)
	for _, v := range s.Result {
		mp[strings.Split(v.Name, ".")[0]] = v.Id
	}
	return mp
}

// 更新域名
func (c *CloudflareAPI) UpdateDomain(ip []string) {
	if len(ip) < len(c.SubDomain) {
		fmt.Println("ip地址和域名数量不匹配")
		return
	}
	c.ReadYaml()
	s := c.GetDomainUuid()
	for i, v := range c.SubDomain {
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
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {

			}
		}(res.Body)
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
			fmt.Println("域名:" + v + c.Domain + "更新失败 IP:" + ip[i])
		} else {
			fmt.Println("域名:" + v + c.Domain + "更新成功 IP:" + ip[i])
		}
	}
}
