package process

// Global holds the global configuration.
type Global struct {
	CheckNewVersion    bool `json:"checkNewVersion"`
	SendAnonymousUsage bool `json:"sendAnonymousUsage"`
}

// Log configure logging
type Log struct {
	Level string `json:"level,omitempty"`
}

// ServersTransport options to configure communication between Traefik and the servers.
type ServersTransport struct {
	ForwardingTimeouts STForwardingTimeouts `json:"forwardingTimeouts"`
}

// STForwardingTimeouts define timeouts
type STForwardingTimeouts struct {
	DialTimeout           string `json:"dialTimeout"`
	ResponseHeaderTimeout string `json:"responseHeaderTimeout"`
}

// EntryPoint holds EntryPoint config
type EntryPoint struct {
	Address string `json:"address"`
}

// API holds the API configuration.
type API struct {
	Insecure  bool `json:"insecure"`
	Dashboard bool `json:"dashboard"`
	Debug     bool `json:"debug"`
}

// API holds the API configuration.
type Pilot struct {
	Token     bool `json:"token"`
	Dashboard bool `json:"dashboard,omitempty"`
}

// Metrics provides options to expose and send Traefik metrics to different third party monitoring systems.
type Metrics struct {
	Prometheus *Prometheus `description:"Prometheus metrics exporter type." json:"prometheus,omitempty" export:"true" label:"allowEmpty"`
}

// Prometheus can contain specific configuration used by the Prometheus Metrics exporter.
type Prometheus struct {
	Buckets              []float64 `json:"buckets"`
	AddEntryPointsLabels bool      `json:"addEntryPointsLabels"`
	AddServicesLabels    bool      `json:"addServicesLabels"`
	EntryPoint           string    `json:"entryPoint"`
}

// Kubernetes CRD provider holds configurations of the provider.
type KubernetesCRDProvider struct {
	Endpoint                  string   `description:"Kubernetes server endpoint (required for external cluster client)." json:"endpoint,omitempty" toml:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Token                     string   `description:"Kubernetes bearer token (not needed for in-cluster client)." json:"token,omitempty" toml:"token,omitempty" yaml:"token,omitempty"`
	CertAuthFilePath          string   `description:"Kubernetes certificate authority file path (not needed for in-cluster client)." json:"certAuthFilePath,omitempty" toml:"certAuthFilePath,omitempty" yaml:"certAuthFilePath,omitempty"`
	Namespaces                []string `description:"Kubernetes namespaces." json:"namespaces,omitempty" toml:"namespaces,omitempty" yaml:"namespaces,omitempty" export:"true"`
	AllowCrossNamespace       bool     `description:"Allow cross namespace resource reference." json:"allowCrossNamespace,omitempty" toml:"allowCrossNamespace,omitempty" yaml:"allowCrossNamespace,omitempty" export:"true"`
	AllowExternalNameServices bool     `description:"Allow ExternalName services." json:"allowExternalNameServices,omitempty" toml:"allowExternalNameServices,omitempty" yaml:"allowExternalNameServices,omitempty" export:"true"`
	LabelSelector             string   `description:"Kubernetes label selector to use." json:"labelSelector,omitempty" toml:"labelSelector,omitempty" yaml:"labelSelector,omitempty" export:"true"`
	IngressClass              string   `description:"Value of kubernetes.io/ingress.class annotation to watch for." json:"ingressClass,omitempty" toml:"ingressClass,omitempty" yaml:"ingressClass,omitempty" export:"true"`
}

// EndpointIngress holds the endpoint information for the Kubernetes provider.
type EndpointIngress struct {
	IP               string `description:"IP used for Kubernetes Ingress endpoints." json:"ip,omitempty" toml:"ip,omitempty" yaml:"ip,omitempty"`
	Hostname         string `description:"Hostname used for Kubernetes Ingress endpoints." json:"hostname,omitempty" toml:"hostname,omitempty" yaml:"hostname,omitempty"`
	PublishedService string `description:"Published Kubernetes Service to copy status from." json:"publishedService,omitempty" toml:"publishedService,omitempty" yaml:"publishedService,omitempty"`
}

// Kubernetes Ingress provider holds configurations of the provider.
type KubernetesIngressProvider struct {
	Endpoint                  string           `description:"Kubernetes server endpoint (required for external cluster client)." json:"endpoint,omitempty" toml:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	Token                     string           `description:"Kubernetes bearer token (not needed for in-cluster client)." json:"token,omitempty" toml:"token,omitempty" yaml:"token,omitempty"`
	CertAuthFilePath          string           `description:"Kubernetes certificate authority file path (not needed for in-cluster client)." json:"certAuthFilePath,omitempty" toml:"certAuthFilePath,omitempty" yaml:"certAuthFilePath,omitempty"`
	Namespaces                []string         `description:"Kubernetes namespaces." json:"namespaces,omitempty" toml:"namespaces,omitempty" yaml:"namespaces,omitempty" export:"true"`
	LabelSelector             string           `description:"Kubernetes Ingress label selector to use." json:"labelSelector,omitempty" toml:"labelSelector,omitempty" yaml:"labelSelector,omitempty" export:"true"`
	IngressClass              string           `description:"Value of kubernetes.io/ingress.class annotation or IngressClass name to watch for." json:"ingressClass,omitempty" toml:"ingressClass,omitempty" yaml:"ingressClass,omitempty" export:"true"`
	IngressEndpoint           *EndpointIngress `description:"Kubernetes Ingress Endpoint." json:"ingressEndpoint,omitempty" toml:"ingressEndpoint,omitempty" yaml:"ingressEndpoint,omitempty" export:"true"`
	AllowEmptyServices        bool             `description:"Allow creation of services without endpoints." json:"allowEmptyServices,omitempty" toml:"allowEmptyServices,omitempty" yaml:"allowEmptyServices,omitempty" export:"true"`
	AllowExternalNameServices bool             `description:"Allow ExternalName services." json:"allowExternalNameServices,omitempty" toml:"allowExternalNameServices,omitempty" yaml:"allowExternalNameServices,omitempty" export:"true"`
}

// Providers holds providers configuration
type Providers struct {
	ProvidersThrottleDuration string                    `json:"providersThrottleDuration"`
	File                      map[string]interface{}    `json:"file"`
	KubernetesIngress         KubernetesIngressProvider `json:"kubernetesIngress"`
	KubernetesCRD             KubernetesCRDProvider     `json:"kubernetesCRD,omitempty"`
}

// Router holds the router config
type Router struct {
	EntryPoints []string `json:"entryPoints"`
	Middlewares []string `json:"middlewares,omitempty"`
	Service     string   `json:"service"`
	Rule        string   `json:"rule"`
}

/*   middlewares:
#   admin-auth:
#     basicAuth:
#       users:
#       - admin:$apr1$H6uskkkW$IgXLP6ewTrSuBkTrqE8wj/
*/

// Service configure backend services
type Service struct {
	LoadBalancer TLoadBalancer `json:"loadBalancer"`
}

// TLoadBalancer configure loadbalancing
type TLoadBalancer struct {
	Servers        []Server `json:"servers"`
	PassHostHeader bool     `json:"passHostHeader"`
}

// Server configure backend server
type Server struct {
	URL string `json:"url"`
}

// Http configure Http
type Http struct {
	Routers  map[string]Router  `json:"routers,omitempty"`
	Services map[string]Service `json:"services,omitempty"`
}

// TraefikConfig holds the global configuration
type TraefikConfig struct {
	Global           Global                `json:"global,omitempty"`
	Log              *Log                  `json:"log,omitempty"`
	ServersTransport ServersTransport      `json:"serversTransport,omitempty"`
	EntryPoints      map[string]EntryPoint `json:"entryPoints,omitempty"`
	API              API                   `json:"api,omitempty"`
	Metrics          Metrics               `json:"metrics,omitempty"`
	Providers        Providers             `json:"providers,omitempty"`
	Http             Http                  `json:"http,omitempty"`
	Pilot            Pilot                 `json:"pilot,omitempty"`
}

// NewTraefikConfig constructs a new configuration for traefik loadbalancer
func NewTraefikConfig() TraefikConfig {
	return TraefikConfig{
		Global: Global{
			CheckNewVersion:    false,
			SendAnonymousUsage: false,
		},
		Log: &Log{
			Level: "DEBUG",
		},
		API: API{
			Dashboard: true,
			Debug:     true,
			Insecure:  true,
		},
		EntryPoints: map[string]EntryPoint{
			"primary": {
				Address: ":8090",
			},
			"secondary": {
				Address: ":8091",
			},
			"metrics": {
				Address: ":8092",
			},
		},
		Metrics: Metrics{
			Prometheus: &Prometheus{
				AddEntryPointsLabels: true,
				Buckets:              []float64{0.1, 0.3, 1.2, 5.0},
				AddServicesLabels:    true,
				EntryPoint:           "metrics",
			},
		},
		Providers: Providers{
			ProvidersThrottleDuration: "120s",
			File: map[string]interface{}{
				"watch":    true,
				"filename": "/etc/traefik/traefik.yaml",
			},
			KubernetesIngress: KubernetesIngressProvider{
				IngressClass: "kom-operator",
			},
			KubernetesCRD: KubernetesCRDProvider{
				IngressClass: "kom-operator",
			},
		},
		ServersTransport: ServersTransport{
			ForwardingTimeouts: STForwardingTimeouts{
				DialTimeout:           "300s",
				ResponseHeaderTimeout: "300s",
			},
		},
		Pilot: Pilot{
			Dashboard: false,
		},
	}
}
