package plugins

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	whale "github.com/docker/docker/client"
	"github.com/kahgeh/envoyXds/utility"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Docker provides configuration source from docker
type Docker struct {
}

const (
	docker      = "Docker"
	commitIDKey = "COMMIT_ID"
	versionKey  = "VERSION"
)

var (
	portGroupExpr          = "(?P<port>\\d+)"
	urlPrefixExpr          = fmt.Sprintf("CLUSTER_%s_URLPREFIX", portGroupExpr)
	urlPrefixPattern       = regexp.MustCompile(urlPrefixExpr)
	serviceNameExpr        = fmt.Sprintf("CLUSTER_%s_NAME", portGroupExpr)
	serviceNamePattern     = regexp.MustCompile(serviceNameExpr)
	serviceCategoryExpr    = fmt.Sprintf("CLUSTER_%s_CATEGORY", portGroupExpr)
	serviceCategoryPattern = regexp.MustCompile(serviceCategoryExpr)
)

var (
	serviceNameSubmatchGroupLookup = utility.ToMap(serviceNamePattern.SubexpNames())
	portIndex                      = serviceNameSubmatchGroupLookup["port"]
)

type service struct {
	name      string
	category  string
	urlPrefix string
	version   string
	port      uint16
}

type discoverableContainer struct {
	services []service
	types.Container
}

func getServicePorts(container types.Container) []uint16 {
	ports := []uint16{}
	uniquePortsContainer := make(map[uint16]string)
	for key := range container.Labels {
		if serviceNamePattern.MatchString(key) {
			submatches := serviceNamePattern.FindStringSubmatch(key)
			port := uint16(utility.MustAtoi(submatches[portIndex]))
			if _, alreadyExists := uniquePortsContainer[port]; !alreadyExists {
				ports = append(ports, port)
				uniquePortsContainer[port] = "exist"
			}
		}
	}
	return ports
}

func mapContainerToDiscoverableContainer(container types.Container, servicePorts []uint16) *discoverableContainer {
	labels := container.Labels
	discoveredContainer := &discoverableContainer{
		Container: container,
	}
	services := []service{}
	for _, port := range servicePorts {
		serviceNameLabelKey := strings.Replace(serviceNameExpr, portGroupExpr, strconv.Itoa(int(port)), 1)
		serviceCategoryLabelKey := strings.Replace(serviceCategoryExpr, portGroupExpr, strconv.Itoa(int(port)), 1)
		urlPrefixLabelKey := strings.Replace(urlPrefixExpr, portGroupExpr, strconv.Itoa(int(port)), 1)

		service := service{
			name:      labels[serviceNameLabelKey],
			category:  labels[serviceCategoryLabelKey],
			urlPrefix: labels[urlPrefixLabelKey],
			version:   fmt.Sprintf("v%s-%s", labels[versionKey], labels[commitIDKey]),
			port:      port,
		}
		services = append(services, service)
	}
	discoveredContainer.services = services
	return discoveredContainer
}

func getDiscoverableContainers(containers []types.Container) *[]discoverableContainer {
	discoveredContainers := []discoverableContainer{}
	for _, container := range containers {
		servicePorts := getServicePorts(container)
		if len(servicePorts) > 0 {
			discoveredContainers = append(discoveredContainers,
				*mapContainerToDiscoverableContainer(container, servicePorts))
		}
	}
	return &discoveredContainers
}

type enPorts []types.Port

func (ports enPorts) wherePorts(predicate func(types.Port) bool) []types.Port {
	matchingPorts := []types.Port{}
	for _, port := range ports {
		if predicate(port) {
			matchingPorts = append(matchingPorts, port)
		}
	}
	return matchingPorts
}

func (ports enPorts) getMappedAddress(portNumber uint16) (mappedHost string, mappedPortNumber uint16) {
	mappedPorts := ports.
		wherePorts(func(p types.Port) bool {
			return p.PrivatePort == portNumber
		})
	if len(mappedPorts) > 0 {
		mappedHost = "host.docker.internal"
		mappedPortNumber = mappedPorts[0].PublicPort
	}
	return
}

func (container *discoverableContainer) mapToEndpointUpdateRequest() *EndpointUpdateRequest {
	endpoints := []Endpoint{}
	for _, service := range container.services {
		host, portNumber := enPorts(container.Ports).
			getMappedAddress(service.port)

		frontProxyPath := fmt.Sprintf("/%s/%s", service.category,
			service.name)
		if service.urlPrefix != "" {
			frontProxyPath = fmt.Sprintf("/%s/%s", service.category,
				service.urlPrefix)
		}

		endpoint := Endpoint{
			UniqueID:       container.ID,
			Clustername:    service.name,
			Host:           host,
			Port:           portNumber,
			FrontProxyPath: frontProxyPath,
			Version:        service.version,
		}
		endpoints = append(endpoints, endpoint)
	}

	request := &EndpointUpdateRequest{
		PluginName: docker,
		Timestamp:  time.Now(),
		Endpoints:  endpoints,
	}

	return request
}

func (plugin *Docker) getName() string {
	return docker
}

func (plugin *Docker) run(ctx context.Context) chan *EndpointUpdateRequest {
	cli, err := whale.NewEnvClient()
	if err != nil {
		panic(err)
	}
	channel := make(chan *EndpointUpdateRequest)
	go func(plugin *Docker) {
		defer close(channel)
		for {

			containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
			if err != nil {
				panic(err)
			}

			discoveredContainers := getDiscoverableContainers(containers)

			for _, container := range *discoveredContainers {
				select {
				case channel <- container.mapToEndpointUpdateRequest():
					fmt.Printf("[%s] sending update request\n", plugin.getName())
				case <-ctx.Done():
					fmt.Printf("[%s] terminating scanner loop\n", plugin.getName())
					return
				}

			}

			select {
			case <-time.After(2 * time.Second):
				fmt.Printf("[%s] going again\n", plugin.getName())
			case <-ctx.Done():
				fmt.Printf("[%s] terminating scanner loop\n", plugin.getName())
				return
			}
		}
	}(plugin)
	return channel
}
