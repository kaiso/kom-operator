package process

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"sigs.k8s.io/yaml"
)

var mutex *sync.Mutex = &sync.Mutex{}

// CreateRouter creates a router in the loadBalancer
func (lb *LoadBalancer) CreateRouter(namespace string, service string, routes map[int32]string) error {

	mutex.Lock()
	defer mutex.Unlock()

	lbLogger.Info(fmt.Sprintf("Creating  a router for service [%v] in namespace [%v]", service, namespace))
	err := lb.loadConfig()
	if err != nil {
		lbLogger.Error(err, "Fatal, unable to create new router, problem with the loadbalancer configuration")
		return err
	}

	rawconfig := lb.Config.Data[traefikConfigFileName]

	var cfg *TraefikConfig

	yaml.Unmarshal([]byte(rawconfig), &cfg)

	for port, rule := range routes {
		var url strings.Builder
		var serviceFound bool = false
		var routerFound bool = false
		var routerName string
		if !strings.Contains(service, "@internal") {
			routerName = namespace + "_" + service + "_" + strconv.Itoa(int(port))
			url.WriteString("http://")
			url.WriteString(service)
			url.WriteString(".")
			url.WriteString(namespace)
			url.WriteString(".svc.cluster.local:")
			if port != 0 {
				url.WriteString(strconv.Itoa(int(port)))
			} else {
				url.WriteString(strconv.Itoa(80))
			}
		} else {
			serviceFound = true
			routerName = service
		}

		if !serviceFound {
			for key := range cfg.Http.Services {
				if key == routerName {
					serviceFound = true
					break
				}
			}
		}

		if !serviceFound {
			if cfg.Http.Services == nil {
				cfg.Http.Services = make(map[string]Service)
			}
			cfg.Http.Services[routerName] = Service{
				LoadBalancer: TLoadBalancer{
					Servers: []Server{{
						URL: url.String(),
					},
					},
					PassHostHeader: true,
				},
			}
		}

		for key := range cfg.Http.Routers {
			if key == routerName {
				routerFound = true
				break
			}
		}

		if !routerFound {
			if cfg.Http.Routers == nil {
				cfg.Http.Routers = make(map[string]Router)
			}
			cfg.Http.Routers[routerName] = Router{
				Service:     routerName,
				EntryPoints: []string{"primary"},
				Rule:        rule,
			}
		}
	}

	err = lb.writeConfig(*cfg, true)
	if err != nil {
		return err
	}

	err = lb.reloadDeployment()

	if err != nil {
		return err
	}

	lbLogger.Info(fmt.Sprintf("Configured router for %+v", service))
	return nil
}

// RemoveRouter removes a router from the loadBalancer
func (lb *LoadBalancer) RemoveRouter(namespace string, service string, reload bool) error {

	mutex.Lock()
	defer mutex.Unlock()

	lbLogger.Info(fmt.Sprintf("Removing all routers for service [%v] in namespace [%v]", service, namespace))
	err := lb.loadConfig()
	if err != nil {
		lbLogger.Error(err, "Fatal, unable to load the loadbalancer configuration")
		return err
	}

	rawconfig := lb.Config.Data[traefikConfigFileName]

	var cfg *TraefikConfig

	yaml.Unmarshal([]byte(rawconfig), &cfg)

	var routerName string = service
	if !strings.Contains(service, "@internal") {
		routerName = namespace + "_" + service + "_"
		for key := range cfg.Http.Services {
			if strings.Contains(key, routerName) {
				delete(cfg.Http.Services, key)
				for key := range cfg.Http.Routers {
					if strings.Contains(key, routerName) {
						delete(cfg.Http.Routers, key)
					}
				}
			}
		}
	}

	err = lb.writeConfig(*cfg, true)
	if err != nil {
		return err
	}

	if reload == true {
		err = lb.reloadDeployment()
		if err != nil {
			return err
		}
	}

	lbLogger.Info(fmt.Sprintf("Removed router for %+v", service))
	return nil
}
