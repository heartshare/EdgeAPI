package dnsclients

import "github.com/iwind/TeaGo/maps"

type ProviderType = string

// 服务商代号
const (
	ProviderTypeDNSPod     ProviderType = "dnspod"
	ProviderTypeAliDNS     ProviderType = "alidns"
	ProviderTypeDNSCom     ProviderType = "dnscom"
	ProviderTypeCustomHTTP ProviderType = "customHTTP"
)

// 所有的服务商类型
var AllProviderTypes = []maps.Map{
	{
		"name": "阿里云DNS",
		"code": ProviderTypeAliDNS,
	},
	{
		"name": "DNSPod",
		"code": ProviderTypeDNSPod,
	},
	/**{
		"name": "帝恩思DNS.COM",
		"code": ProviderTypeDNSCom,
	},**/
	{
		"name": "自定义HTTP DNS",
		"code": ProviderTypeCustomHTTP,
	},
}

// 查找服务商实例
func FindProvider(providerType ProviderType) ProviderInterface {
	switch providerType {
	case ProviderTypeDNSPod:
		return &DNSPodProvider{}
	case ProviderTypeAliDNS:
		return &AliDNSProvider{}
	case ProviderTypeCustomHTTP:
		return &CustomHTTPProvider{}
	}
	return nil
}

// 查找服务商名称
func FindProviderTypeName(providerType ProviderType) string {
	for _, t := range AllProviderTypes {
		if t.GetString("code") == providerType {
			return t.GetString("name")
		}
	}
	return ""
}
