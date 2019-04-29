package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	plugins "github.com/kahgeh/envoyXds/plugins"
	myals "github.com/kahgeh/envoyXds/server/accesslogs"
	"sync"
	"time"

	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	//"github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	lis "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	accesslog "github.com/envoyproxy/go-control-plane/envoy/service/accesslog/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"

	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	"github.com/envoyproxy/go-control-plane/pkg/util"
)

var (
	debug       bool
	onlyLogging bool

	localhost = "127.0.0.1"

	port        uint
	gatewayPort uint
	alsPort     uint

	mode string

	version int32

	config cache.SnapshotCache
)

const (
	XdsCluster = "xds_cluster"
	Ads        = "ads"
	Xds        = "xds"
	Rest       = "rest"
)

func init() {
	flag.BoolVar(&debug, "debug", true, "Use debug logging")
	flag.BoolVar(&onlyLogging, "onlyLogging", false, "Only demo AccessLogging Service")
	flag.UintVar(&port, "port", 18000, "Management server port")
	flag.UintVar(&gatewayPort, "gateway", 18001, "Management server port for HTTP gateway")
	flag.UintVar(&alsPort, "als", 18090, "Accesslog server port")
	flag.StringVar(&mode, "ads", Ads, "Management server type (ads, xds, rest)")
}

type logger struct{}

func (logger logger) Infof(format string, args ...interface{}) {
	log.Debugf(format, args...)
}
func (logger logger) Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

type callbacks struct {
	signal   chan struct{}
	fetches  int
	requests int
	mu       sync.Mutex
}

func (cb *callbacks) Report() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	log.WithFields(log.Fields{"fetches": cb.fetches, "requests": cb.requests}).Info("server callbacks")
}
func (cb *callbacks) OnStreamOpen(_ context.Context, id int64, typ string) error {
	log.Debugf("stream %d open for %s", id, typ)
	return nil
}
func (cb *callbacks) OnStreamClosed(id int64) {
	log.Debugf("stream %d closed", id)
}
func (cb *callbacks) OnStreamRequest(int64, *v2.DiscoveryRequest) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.requests++
	if cb.signal != nil {
		close(cb.signal)
		cb.signal = nil
	}
	return nil
}
func (cb *callbacks) OnStreamResponse(int64, *v2.DiscoveryRequest, *v2.DiscoveryResponse) {
	log.Infof("OnStreamResponse...")
	cb.Report()
}
func (cb *callbacks) OnFetchRequest(_ context.Context, req *v2.DiscoveryRequest) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.fetches++
	if cb.signal != nil {
		close(cb.signal)
		cb.signal = nil
	}
	return nil
}
func (cb *callbacks) OnFetchResponse(*v2.DiscoveryRequest, *v2.DiscoveryResponse) {}

// Hasher returns node ID as an ID
type Hasher struct {
}

// ID function
func (h Hasher) ID(node *core.Node) string {
	if node == nil {
		return "unknown"
	}
	return node.Id
}

// RunAccessLogServer starts an accesslog service.
func RunAccessLogServer(ctx context.Context, als *myals.AccessLogService, port uint) {
	grpcServer := grpc.NewServer()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.WithError(err).Fatal("failed to listen")
	}

	accesslog.RegisterAccessLogServiceServer(grpcServer, als)
	log.WithFields(log.Fields{"port": port}).Info("access log server listening")

	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			log.Error(err)
		}
	}()
	<-ctx.Done()

	grpcServer.GracefulStop()
}

const grpcMaxConcurrentStreams = 1000000

// RunManagementServer starts an xDS server at the given port.
func RunManagementServer(ctx context.Context, server xds.Server, port uint) {
	// gRPC golang library sets a very small upper bound for the number gRPC/h2
	// streams over a single TCP connection. If a proxy multiplexes requests over
	// a single connection to the management server, then it might lead to
	// availability problems.
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams))
	grpcServer := grpc.NewServer(grpcOptions...)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.WithError(err).Fatal("failed to listen")
	}

	// register services
	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	v2.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	v2.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	v2.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	v2.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	discovery.RegisterSecretDiscoveryServiceServer(grpcServer, server)

	log.WithFields(log.Fields{"port": port}).Info("management server listening")
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			log.Error(err)
		}
	}()
	<-ctx.Done()

	grpcServer.GracefulStop()
}

// RunManagementGateway starts an HTTP gateway to an xDS server.
func RunManagementGateway(ctx context.Context, srv xds.Server, port uint) {
	log.WithFields(log.Fields{"port": port}).Info("gateway listening HTTP/1.1")
	server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: &xds.HTTPGateway{Server: srv}}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Error(err)
		}
	}()
}

func main() {
	flag.Parse()
	if debug {
		log.SetLevel(log.DebugLevel)
	}
	// main thread context
	ctx := context.Background()

	log.Printf("Starting control plane")

	signal := make(chan struct{})
	cb := &callbacks{
		signal:   signal,
		fetches:  0,
		requests: 0,
	}
	config = cache.NewSnapshotCache(mode == Ads, Hasher{}, logger{})

	srv := xds.NewServer(config, cb)

	//als := &accesslogs.AccessLogService{}
	als := &myals.AccessLogService{}
	go RunAccessLogServer(ctx, als, alsPort)

	if onlyLogging {
		cc := make(chan struct{})
		<-cc
		os.Exit(0)
	}

	// start the xDS server
	go RunManagementServer(ctx, srv, port)
	go RunManagementGateway(ctx, srv, gatewayPort)

	fmt.Printf("some signals from server %s\n", <-signal)

	als.Dump(func(s string) { log.Debug(s) })
	cb.Report()
	dockerPlugin := &plugins.Docker{}
	allPlugins := []plugins.Plugin{dockerPlugin}
	channel := plugins.RunAllPlugins(ctx, allPlugins)
	existingIds := make(map[string]string)

	for {
		select {
		case updateRequest := <-channel:
			endpoint := (*updateRequest).Endpoints[0]
			_, alreadyExists := existingIds[endpoint.UniqueID]
			if !alreadyExists {
				fmt.Printf("%v\n", updateRequest)
				nodeId := config.GetStatusKeys()[0]
				clusterName := endpoint.Clustername
				h := &core.Address{Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Address:  endpoint.Host,
						Protocol: core.TCP,
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: uint32(endpoint.Port),
						},
					},
				}}

				c := []cache.Resource{
					&v2.Cluster{
						Name:                 clusterName,
						ConnectTimeout:       2 * time.Second,
						ClusterDiscoveryType: &v2.Cluster_Type{Type: v2.Cluster_STRICT_DNS},
						DnsLookupFamily:      v2.Cluster_V4_ONLY,
						LbPolicy:             v2.Cluster_ROUND_ROBIN,
						Hosts:                []*core.Address{h},
					},
				}

				var listenerName = "listener_0"
				//var targetHost = endpoint.Host
				var targetPath = endpoint.FrontProxyPath
				var virtualHostName = "local_service"
				var routeConfigName = "local_route"

				v := route.VirtualHost{
					Name:    virtualHostName,
					Domains: []string{"*"},

					Routes: []route.Route{{
						Match: route.RouteMatch{
							PathSpecifier: &route.RouteMatch_Path{
								Path: targetPath,
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
					}}}

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

				var l = []cache.Resource{
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
				snap := cache.NewSnapshot(fmt.Sprint(version), nil, c, nil, l)

				config.SetSnapshot(nodeId, snap)
			}

		case <-ctx.Done():
			return
		}

		fmt.Scanln()
	}
}