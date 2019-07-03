package nacos

import (
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/kelseyhightower/confd/log"
	"fmt"
	"strings"
	"net/url"
	"strconv"
)

var replacer = strings.NewReplacer("/", ".")

type Client struct {
	client config_client.IConfigClient
	group string
	namespace string
	accessKey string
	secretKey string
	channel chan int
}

func NewNacosClient(nodes []string, group string, config constant.ClientConfig) (client *Client, err error) {
	var configClient config_client.IConfigClient
	servers := []constant.ServerConfig{
	}
	for _, key := range nodes {
		nacosUrl,_ := url.Parse(key)

		port, _ := strconv.Atoi(nacosUrl.Port())
		servers = append(servers, constant.ServerConfig{
			IpAddr: nacosUrl.Hostname(),
			Port:   uint64(port),
		})
	}

	if len(strings.TrimSpace(group)) == 0 {
		group = "DEFAULT_GROUP"
	}

	log.Info(fmt.Sprintf("endpoint=%s, namespace=%s, group=%s, accessKey=%s, secretKey=%s", config.Endpoint, config.NamespaceId, group, config.AccessKey, config.SecretKey))

	configClient, err = clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": servers,
		"clientConfig": constant.ClientConfig{
			TimeoutMs:           20000,
			ListenInterval:      10000,
			NotLoadCacheAtStart: true,
			NamespaceId:	     config.NamespaceId,
			AccessKey: 			 config.AccessKey,
			SecretKey: 			 config.SecretKey,
			Endpoint:   		 config.Endpoint,
		},
	})

	client = &Client{configClient, group, config.NamespaceId, config.AccessKey, config.SecretKey, make(chan int)}

	return
}

func (client *Client) GetValues(keys []string) (map[string]string, error) {
	vars := make(map[string]string)
	for _, key := range keys {
		k := strings.TrimPrefix(key, "/")
		k = replacer.Replace(k)
		resp, err := client.client.GetConfig(vo.ConfigParam{
			DataId:  k,
			Group: client.group,
		})
		log.Info(fmt.Sprintf("key=%s, value=%s", key, resp))
		if err == nil {
			vars[key] = resp
		}
	}

	return vars, nil
}

func (client *Client) WatchPrefix(prefix string, keys []string, waitIndex uint64, stopChan chan bool) (uint64, error) {
	// return something > 0 to trigger a key retrieval from the store
	if waitIndex == 0 {
		for _, key := range keys {
			k := strings.TrimPrefix(key, "/")
			k = replacer.Replace(k)

			err := client.client.ListenConfig(vo.ConfigParam{
				DataId:  k,
				Group: client.group,
				OnChange: func(namespace, group, dataId, data string) {
					log.Info(fmt.Sprintf("config namespace=%s, dataId=%s, group=%s has changed", namespace, dataId, group))
					client.channel <- 1
				},
			})

			if err != nil {
				return 0,err
			}
		}

		return 1, nil
	}

	select {
		case <- client.channel:
			return waitIndex,nil

	}

	return waitIndex, nil
}
