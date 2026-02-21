package cnpg

import (
	"context"
	"fmt"

	"github.com/cnpg-broker/pkg/catalog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
	_, err := c.dynamic.Resource(clusterResource).Namespace(instanceId).Get(ctx, instanceId, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	host := fmt.Sprintf("%s-rw.%s.svc", instanceId, instanceId)

	// TODO: fix this entire func, Claude just produced hilarious garbage! 🤣
	return map[string]string{
		"host":     host,
		"port":     "5432",
		"database": "app",
		"username": "app",
		"password": fmt.Sprintf("secret://%s-app", instanceId),
		"uri":      fmt.Sprintf("postgresql://app@%s:5432/app", host),
	}, nil
}
