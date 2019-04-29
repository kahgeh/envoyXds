package plugins

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Endpoint represent the service endpoint
type Endpoint struct {
	UniqueID       string
	Clustername    string
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
