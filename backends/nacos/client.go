package nacos

import (
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/example"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/utils"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"log"
	"time"
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

		fmt.Println(key)
		fmt.Println(nacosUrl.Hostname())
		fmt.Println(nacosUrl.Port())
		port, _ := strconv.Atoi(nacosUrl.Port())
		servers = append(servers, constant.ServerConfig{
			IpAddr: nacosUrl.Hostname(),
			Port:   uint64(port),
		})
	}

	fmt.Println("namespace=" + config.NamespaceId)
	fmt.Println("AccessKey=" + config.AccessKey)
	fmt.Println("SecretKey=" + config.SecretKey)
	fmt.Println("Endpoint=" + config.Endpoint)

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

	if len(strings.TrimSpace(group)) == 0 {
		group = "DEFAULT_GROUP"
	}

	namespace := strings.TrimSpace(config.NamespaceId)


	client = &Client{configClient, group, namespace, config.AccessKey, config.SecretKey, make(chan int)}
	fmt.Println("hello nacos")

	return
}

func (client *Client) GetValues(keys []string) (map[string]string, error) {
	vars := make(map[string]string)
	fmt.Println(keys)
	for _, key := range keys {
		k := strings.TrimPrefix(key, "/")
		k = replacer.Replace(k)
		resp, err := client.client.GetConfig(vo.ConfigParam{
			DataId:  k,
			Group: client.group,
		})
		fmt.Println(k + ":" + resp)
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
					fmt.Println(data)
					client.channel <- 1
				},
			})
			fmt.Println(key)

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
	fmt.Print("waitIndex=")
	fmt.Println(waitIndex)
	fmt.Println(prefix)
	fmt.Println(keys)

	return waitIndex, nil
}

func main() {

	configClient, _ := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": []constant.ServerConfig{
			{
				IpAddr: "127.0.0.1",
				Port:   8848,
			},
		},
		"clientConfig": constant.ClientConfig{
			TimeoutMs:           20000,
			ListenInterval:      10000,
			NotLoadCacheAtStart: true,
		},
	})

	content, err := configClient.GetConfig(vo.ConfigParam{
		DataId:  "com.alibaba",
		Group: "tsing",
		Content: "123",
		Tag:	"",
		AppName: "123",
	})

	if err == nil {
		fmt.Print(content)
	} else {
		fmt.Print(content)
		fmt.Print(err)
	}

	configClient.ListenConfig(vo.ConfigParam{
		DataId:  "com.alibaba",
		Group: "tsing",
		OnChange: func(namespace, group, dataId, data string) {
			fmt.Println("")
		},
	})


	client, _ := clients.CreateNamingClient(map[string]interface{}{
		"serverConfigs": []constant.ServerConfig{
			{
				IpAddr: "127.0.0.1",
				Port:   8848,
			},
		},
		"clientConfig": constant.ClientConfig{
			TimeoutMs:           20000,
			ListenInterval:      10000,
			NotLoadCacheAtStart: true,
		},
	})

	example.ExampleServiceClient_RegisterServiceInstance(client, vo.RegisterInstanceParam{
		Ip:          "127.0.0.1",
		Port:        8848,
		ServiceName: "demo.go",
		Weight:      10,
		ClusterName: "a",
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
	})

	example.ExampleServiceClient_GetService(client)
	param := &vo.SubscribeParam{
		ServiceName: "demo.go",
		Clusters:    []string{"a"},
		SubscribeCallback: func(services []model.SubscribeService, err error) {
			log.Printf("\n\n callback return services:%s \n\n", utils.ToJsonString(services))
		},
	}
	example.ExampleServiceClient_Subscribe(client, param)
	time.Sleep(20 * time.Second)
	example.ExampleServiceClient_RegisterServiceInstance(client, vo.RegisterInstanceParam{
		Ip:          "127.0.0.1",
		Port:        8848,
		ServiceName: "demo.go",
		Weight:      10,
		ClusterName: "a",
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
	})
	time.Sleep(20 * time.Second)
	example.ExampleServiceClient_UnSubscribe(client, param)
	example.ExampleServiceClient_DeRegisterServiceInstance(client, vo.DeregisterInstanceParam{
		Ip:          "127.0.0.1",
		Ephemeral:   true,
		Port:        8848,
		ServiceName: "demo.go",
		Cluster:     "a",
	})

}
