package registryplugins

import (
	"context"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"sync"
	"time"
)

// Endpoint represent the service endpoint
type Endpoint struct {
	UniqueID       string
	ClusterName    string
	Port           uint16
	Host           string
	FrontProxyPath string
	Version        string
}

// EndpointUpdateRequest represent the update request
type EndpointUpdateRequest struct {
	PluginName string
	Timestamp  time.Time
	Endpoints  []Endpoint
}

// Plugin is the extension point for configuration data sources
type Plugin interface {
	getName() string
	run(ctx context.Context) chan *EndpointUpdateRequest
}

func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func (request *EndpointUpdateRequest) GroupByCluster() map[string][]Endpoint {
	clusters := make(map[string][]Endpoint)
	for _, endpoint := range request.Endpoints {
		if _, alreadyExist := clusters[endpoint.ClusterName]; !alreadyExist {
			clusters[endpoint.ClusterName] = []Endpoint{endpoint}
			continue
		}
		endpoints := clusters[endpoint.ClusterName]
		clusters[endpoint.ClusterName] = append(endpoints, endpoint)
	}
	return clusters
}

// GetHash provide an indication if endpoints are the same from another set of endpoints
func (request *EndpointUpdateRequest) GetHash() uint32 {
	ids := []string{}
	for _, endpoint := range request.Endpoints {
		ids = append(ids, endpoint.UniqueID)
	}
	sort.Strings(ids)
	return hash(strings.Join(ids, ""))
}

// RunAllPlugins starts off all the plugins and fan-in all their channels into the returned channel
func RunAllPlugins(ctx context.Context, plugins []Plugin) chan *EndpointUpdateRequest {
	channel := make(chan *EndpointUpdateRequest)
	go func() {
		defer close(channel)
		var waitgroup sync.WaitGroup
		for _, plugin := range plugins {
			waitgroup.Add(1)
			go func(p *Plugin, wg *sync.WaitGroup) {
				defer wg.Done()
				pluginChannel := (*p).run(ctx)
				for {
					request, more := <-pluginChannel
					if !more {
						fmt.Printf("[%s] fan in loop terminating, channel closed\n", (*p).getName())
						return
					}
					channel <- request
				}
			}(&plugin, &waitgroup)
		}
		waitgroup.Wait()
		fmt.Println("[Plugins] All plugins terminated")
	}()
	return channel
}
