package cnpg

import (
	"context"
	"fmt"

	"github.com/cnpg-broker/pkg/catalog"
	"github.com/cnpg-broker/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var clusterResource = schema.GroupVersionResource{
	Group:    "postgresql.cnpg.io",
	Version:  "v1",
	Resource: "clusters",
}

var poolerResource = schema.GroupVersionResource{
	Group:    "postgresql.cnpg.io",
	Version:  "v1",
	Resource: "poolers",
}

type Client struct {
	dynamic   dynamic.Interface
	clientset *kubernetes.Clientset
}

func NewClient() *Client {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
		if err != nil {
			panic(err)
		}
	}

	dynClient := dynamic.NewForConfigOrDie(config)
	clientset := kubernetes.NewForConfigOrDie(config)

	return &Client{
		dynamic:   dynClient,
		clientset: clientset,
	}
}

func (c *Client) CreateCluster(ctx context.Context, instanceId, planId string) (string, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: instanceId,
		},
	}
	_, err := c.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	// Cluster with specs according to planId
	instances, cpu, memory, storage := catalog.PlanSpec(planId)
	cluster := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "postgresql.cnpg.io/v1",
			"kind":       "Cluster",
			"metadata": map[string]any{
				"name":      instanceId,
				"namespace": instanceId,
			},
			"spec": map[string]any{
				"instances": instances,
				"storage": map[string]any{
					"size": storage,
				},
				"resources": map[string]any{
					"requests": map[string]any{
						"cpu":    cpu,
						"memory": memory,
					},
					"limits": map[string]any{
						"cpu":    cpu,
						"memory": memory,
					},
				},
			},
		},
	}

	_, err = c.dynamic.Resource(clusterResource).Namespace(instanceId).Create(ctx, cluster, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	// LoadBalancer service(s), create our own because we'll create multiple of them, with different ports
	lbSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-lb-rw", instanceId),
			Namespace: instanceId,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Name:       "postgres",
					Port:       5432,
					TargetPort: intstr.FromInt(5432),
				},
			},
			Selector: map[string]string{
				"cnpg.io/cluster":      instanceId,
				"cnpg.io/instanceRole": "primary",
			},
		},
	}
	_, err = c.clientset.CoreV1().Services(instanceId).Create(ctx, lbSvc, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	// Pooler for HA clusters
	if instances > 1 {
		pooler := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "postgresql.cnpg.io/v1",
				"kind":       "Pooler",
				"metadata": map[string]any{
					"name":      fmt.Sprintf("%s-pooler", instanceId),
					"namespace": instanceId,
				},
				"spec": map[string]any{
					"cluster": map[string]any{
						"name": instanceId,
					},
					"instances": instances,
					"type":      "rw",
					"pgbouncer": map[string]any{
						"poolMode": "session",
					},
				},
			},
		}
		_, err = c.dynamic.Resource(poolerResource).Namespace(instanceId).Create(ctx, pooler, metav1.CreateOptions{})
		if err != nil {
			return "", err
		}

		// LoadBalancer service for Pooler
		lbSvc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-lb-pooler", instanceId),
				Namespace: instanceId,
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeLoadBalancer,
				Ports: []corev1.ServicePort{
					{
						Name:       "postgres",
						Port:       6432,
						TargetPort: intstr.FromInt(5432),
					},
				},
				Selector: map[string]string{
					"cnpg.io/poolerName": fmt.Sprintf("%s-pooler", instanceId),
				},
			},
		}
		_, err = c.clientset.CoreV1().Services(instanceId).Create(ctx, lbSvc, metav1.CreateOptions{})
		if err != nil {
			return "", err
		}
	}

	return instanceId, nil
}

func (c *Client) GetCluster(ctx context.Context, instanceId string) (*ClusterInfo, error) {
	cluster, err := c.dynamic.Resource(clusterResource).Namespace(instanceId).Get(ctx, instanceId, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	info := &ClusterInfo{
		InstanceID: instanceId,
		Namespace:  instanceId,
		Labels:     cluster.GetLabels(),
	}

	if status, found, err := unstructured.NestedMap(cluster.Object, "status"); found && err == nil {
		if phase, ok := status["phase"].(string); ok {
			info.Phase = phase
		}
		if instances, ok := status["instances"].(int64); ok {
			info.Instances = instances
		}
		if ready, ok := status["readyInstances"].(int64); ok {
			info.Ready = ready
		}
	}

	if info.Ready == info.Instances && info.Instances > 0 {
		info.Status = "ready"
	} else if info.Ready > 0 {
		info.Status = "partially_ready"
	} else {
		info.Status = "not_ready"
	}

	return info, nil
}

func (c *Client) DeleteCluster(ctx context.Context, instanceId string) error {
	return c.clientset.CoreV1().Namespaces().Delete(ctx, instanceId, metav1.DeleteOptions{})
}

func (c *Client) GetCredentials(ctx context.Context, instanceId string) (map[string]string, error) {
	logger.Debug("collecting credentials for instance %s", instanceId)
	
	secretName := fmt.Sprintf("%s-app", instanceId)
	secret, err := c.clientset.CoreV1().Secrets(instanceId).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	host := string(secret.Data["host"])
	username := string(secret.Data["username"])
	password := string(secret.Data["password"])
	database := string(secret.Data["dbname"])
	fqdnUri := string(secret.Data["fqdn-uri"])
	jdbcUri := string(secret.Data["fqdn-jdbc-uri"])
	credentials := map[string]string{
		"host":        host,
		"port":        "5432",
		"database":    database,
		"username":    username,
		"password":    password,
		"uri":         fqdnUri,
		"jdbc_uri":    jdbcUri,
		"ro_host":     fmt.Sprintf("%s-ro", instanceId),
		"ro_uri":      fmt.Sprintf("postgresql://%s:%s@%s-ro.%s.svc.cluster.local:5432/%s", username, password, instanceId, instanceId, database),
		"ro_jdbc_uri": fmt.Sprintf("jdbc:postgresql://%s-ro.%s.svc.cluster.local:5432/%s?password=%s&user=%s", instanceId, instanceId, database, password, username),
	}

	logger.Debug("retrieved base credentials for instance %s", instanceId)

	caCertSecretName := fmt.Sprintf("%s-ca", instanceId)
	caCertSecret, err := c.clientset.CoreV1().Secrets(instanceId).Get(ctx, caCertSecretName, metav1.GetOptions{})
	if err == nil {
		if caCert, ok := caCertSecret.Data["ca.crt"]; ok {
			credentials["ca_cert"] = string(caCert)
			logger.Debug("collected CA certificate for instance %s", instanceId)
		}
	}

	serverCertSecretName := fmt.Sprintf("%s-server", instanceId)
	serverCertSecret, err := c.clientset.CoreV1().Secrets(instanceId).Get(ctx, serverCertSecretName, metav1.GetOptions{})
	if err == nil {
		if tlsCert, ok := serverCertSecret.Data["tls.crt"]; ok {
			credentials["server_cert"] = string(tlsCert)
		}
		if tlsKey, ok := serverCertSecret.Data["tls.key"]; ok {
			credentials["server_key"] = string(tlsKey)
		}
		if len(credentials["server_cert"]) > 0 {
			logger.Debug("collected server certificate for instance %s", instanceId)
		}
	}

	poolerCertSecretName := fmt.Sprintf("%s-pooler", instanceId)
	poolerCertSecret, err := c.clientset.CoreV1().Secrets(instanceId).Get(ctx, poolerCertSecretName, metav1.GetOptions{})
	if err == nil {
		if tlsCert, ok := poolerCertSecret.Data["tls.crt"]; ok {
			credentials["pooler_cert"] = string(tlsCert)
		}
		if tlsKey, ok := poolerCertSecret.Data["tls.key"]; ok {
			credentials["pooler_key"] = string(tlsKey)
		}
		if len(credentials["pooler_cert"]) > 0 {
			logger.Debug("collected pooler certificate for instance %s", instanceId)
		}
	}

	lbSvcName := fmt.Sprintf("%s-lb-rw", instanceId)
	lbSvc, err := c.clientset.CoreV1().Services(instanceId).Get(ctx, lbSvcName, metav1.GetOptions{})
	if err == nil && len(lbSvc.Status.LoadBalancer.Ingress) > 0 {
		lbHost := lbSvc.Status.LoadBalancer.Ingress[0].IP
		if len(lbHost) == 0 {
			lbHost = lbSvc.Status.LoadBalancer.Ingress[0].Hostname
		}
		credentials["lb_host"] = lbHost
		credentials["lb_uri"] = fmt.Sprintf("postgresql://%s:%s@%s:5432/%s", username, password, lbHost, database)
		credentials["lb_jdbc_uri"] = fmt.Sprintf("jdbc:postgresql://%s:5432/%s?password=%s&user=%s", lbHost, database, username, password)

		poolerSvcName := fmt.Sprintf("%s-lb-pooler", instanceId)
		poolerSvc, err := c.clientset.CoreV1().Services(instanceId).Get(ctx, poolerSvcName, metav1.GetOptions{})
		if err == nil && len(poolerSvc.Status.LoadBalancer.Ingress) > 0 {
			lbHost := poolerSvc.Status.LoadBalancer.Ingress[0].IP
			if len(lbHost) == 0 {
				lbHost = poolerSvc.Status.LoadBalancer.Ingress[0].Hostname
			}
			credentials["pooler_host"] = lbHost
			credentials["pooler_port"] = "6432"
			credentials["pooler_uri"] = fmt.Sprintf("postgresql://%s:%s@%s:6432/%s", username, password, lbHost, database)
			credentials["pooler_jdbc_uri"] = fmt.Sprintf("jdbc:postgresql://%s:6432/%s?password=%s&user=%s", lbHost, database, username, password)
		}
	}

	return credentials, nil
}
