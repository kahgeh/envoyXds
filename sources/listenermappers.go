package sources

import (
	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	lis "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	rp "github.com/kahgeh/envoyXds/sources/registryplugins"
)

// MapToListenerResource maps endpoints to listener resources
func MapToListenerResource(clusters map[string][]rp.Endpoint) []cache.Resource {

	var listenerName = "listener_0"
	var routeConfigName = "local_route"

	v := mapToVirtualHost(clusters)

	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.AUTO,
		StatPrefix: "ingress_http",
		RouteSpecifier: &hcm.HttpConnectionManager_RouteConfig{
			RouteConfig: &v2.RouteConfiguration{
				Name:         routeConfigName,
				VirtualHosts: []route.VirtualHost{v},
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: util.Router,
		}},
	}
	pbst, err := util.MessageToStruct(manager)
	if err != nil {
		panic(err)
	}

	return []cache.Resource{
		&v2.Listener{
			Name: listenerName,
			Address: core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Protocol: core.TCP,
						Address:  "0.0.0.0",
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: 80,
						},
					},
				},
			},
			FilterChains: []listener.FilterChain{{
				Filters: []listener.Filter{{
					Name: util.HTTPConnectionManager,
					ConfigType: &lis.Filter_Config{
						Config: pbst,
					},
				}},
			}},
		}}
}

func mapToVirtualHost(clusters map[string][]rp.Endpoint) route.VirtualHost {

	virtualHost := route.VirtualHost{
		Name:    "local_vhost",
		Domains: []string{"*"},
		Routes:  []route.Route{},
	}

	for clusterName, endpoints := range clusters {
		frontProxyPath := endpoints[0].FrontProxyPath
		virtualHost.Routes = append(virtualHost.Routes, mapToRoute(clusterName, frontProxyPath))
	}
	return virtualHost
}

func mapToRoute(clusterName string, frontProxyPath string) route.Route {
	return route.Route{
		Match: route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Path{
				Path: frontProxyPath,
			},
		},
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_Cluster{
					Cluster: clusterName,
				},
				PrefixRewrite: "/",
			},
		},
	}
}
