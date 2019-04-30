package sources

import (
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	rp "github.com/kahgeh/envoyXds/sources/registryplugins"
	"time"
)

// MapToClusterResources maps endpoint to cluster resources
func MapToClusterResources(clusterMap map[string][]rp.Endpoint) []cache.Resource {

	clusters := []v2.Cluster{}
	for clusterName, endpoints := range clusterMap {
		newCluster := createCluster(clusterName)
		for _, endpoint := range endpoints {
			newCluster.Hosts = append(newCluster.Hosts, mapToAddress(endpoint))
		}
		clusters = append(clusters, newCluster)
	}

	resources := []cache.Resource{}
	for _, cluster := range clusters {
		resources = append(resources, &cluster)
	}

	return resources
}

func createCluster(clusterName string) v2.Cluster {

	newCluster := v2.Cluster{
		Name:                 clusterName,
		ConnectTimeout:       2 * time.Second,
		ClusterDiscoveryType: &v2.Cluster_Type{Type: v2.Cluster_STRICT_DNS},
		DnsLookupFamily:      v2.Cluster_V4_ONLY,
		LbPolicy:             v2.Cluster_ROUND_ROBIN,
		Hosts:                []*core.Address{},
	}

	return newCluster
}

func mapToAddress(endpoint rp.Endpoint) *core.Address {
	address := &core.Address{Address: &core.Address_SocketAddress{
		SocketAddress: &core.SocketAddress{
			Address:  endpoint.Host,
			Protocol: core.TCP,
			PortSpecifier: &core.SocketAddress_PortValue{
				PortValue: uint32(endpoint.Port),
			},
		},
	}}

	return address
}
