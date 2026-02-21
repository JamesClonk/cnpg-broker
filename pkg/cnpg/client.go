package cnpg

import (
	"context"
	"fmt"

	"github.com/cnpg-broker/pkg/catalog"
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
			Name:      fmt.Sprintf("%s-rw", instanceId),
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
				"postgresql": instanceId,
				"role":       "primary",
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
					"name":      instanceId,
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
				Name:      fmt.Sprintf("%s-pooler", instanceId),
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
					"cnpg.io/poolerName": instanceId,
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

func (c *Client) GetCluster(ctx context.Context, instanceId string) (string, error) {
	_, err := c.dynamic.Resource(clusterResource).Namespace(instanceId).Get(ctx, instanceId, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	// TODO: return actual cluster here
	return instanceId, nil
}

func (c *Client) DeleteCluster(ctx context.Context, instanceId string) error {
	return c.clientset.CoreV1().Namespaces().Delete(ctx, instanceId, metav1.DeleteOptions{})
}

func (c *Client) GetCredentials(ctx context.Context, instanceId string) (map[string]string, error) {
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

	// get TLS certificates
	tlsSecretName := fmt.Sprintf("%s-ca", instanceId)
	tlsSecret, err := c.clientset.CoreV1().Secrets(instanceId).Get(ctx, tlsSecretName, metav1.GetOptions{})
	if err == nil {
		credentials["ca_cert"] = string(tlsSecret.Data["ca.crt"])
		credentials["tls_cert"] = string(tlsSecret.Data["tls.crt"])
		credentials["tls_key"] = string(tlsSecret.Data["tls.key"])
	}

	// add LB and Pooler endpoints
	lbSvcName := fmt.Sprintf("%s-rw", instanceId)
	lbSvc, err := c.clientset.CoreV1().Services(instanceId).Get(ctx, lbSvcName, metav1.GetOptions{})
	if err == nil && len(lbSvc.Status.LoadBalancer.Ingress) > 0 {
		lbHost := lbSvc.Status.LoadBalancer.Ingress[0].IP
		if len(lbHost) == 0 {
			lbHost = lbSvc.Status.LoadBalancer.Ingress[0].Hostname
		}
		credentials["lb_host"] = lbHost
		credentials["lb_uri"] = fmt.Sprintf("postgresql://%s:%s@%s:5432/%s", username, password, lbHost, database)
		credentials["lb_jdbc_uri"] = fmt.Sprintf("jdbc:postgresql://%s:5432/%s?password=%s&user=%s", lbHost, database, username, password)

		// add Pooler LB if it exists
		poolerSvcName := fmt.Sprintf("%s-pooler", instanceId)
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
