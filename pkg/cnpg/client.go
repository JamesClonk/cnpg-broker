package cnpg

import (
	"context"
	"fmt"

	"github.com/cnpg-broker/pkg/catalog"
	"github.com/google/uuid"
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

func (c *Client) CreateCluster(ctx context.Context, name, planId string) (string, error) {
	namespace := uuid.New().String()
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err := c.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	instances, cpu, memory, disk := catalog.PlanToSpec(planId)
	cluster := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "postgresql.cnpg.io/v1",
			"kind":       "Cluster",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"instances": instances,
				"storage": map[string]interface{}{
					"size": disk,
				},
				"resources": map[string]interface{}{
					"requests": map[string]interface{}{
						"cpu":    cpu,
						"memory": memory,
					},
					"limits": map[string]interface{}{
						"cpu":    cpu,
						"memory": memory,
					},
				},
			},
		},
	}

	_, err = c.dynamic.Resource(clusterResource).Namespace(namespace).Create(ctx, cluster, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}
	return namespace, nil
}

func (c *Client) DeleteCluster(ctx context.Context, namespace string) error {
	return c.clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
}

func (c *Client) GetCredentials(ctx context.Context, name, namespace string) (map[string]string, error) {
	_, err := c.dynamic.Resource(clusterResource).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	host := fmt.Sprintf("%s-rw.%s.svc", name, namespace)

	return map[string]string{
		"host":     host,
		"port":     "5432",
		"database": "app",
		"username": "app",
		"password": fmt.Sprintf("secret://%s-app", name),
		"uri":      fmt.Sprintf("postgresql://app@%s:5432/app", host),
	}, nil
}
