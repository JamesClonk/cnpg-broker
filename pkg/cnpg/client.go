package cnpg

import (
	"context"
	"fmt"
	"strings"

	"github.com/cnpg-broker/pkg/catalog"
	"github.com/cnpg-broker/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

func (c *Client) ListClusters(ctx context.Context) ([]ClusterInfo, error) {
	logger.Debug("listing all clusters")

	namespaces, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: "cnpg-broker.io/instance-id",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	clusters := make([]ClusterInfo, 0, len(namespaces.Items))
	for _, ns := range namespaces.Items {
		instanceId := ns.Name
		clusterInfo, err := c.GetCluster(ctx, instanceId)
		if err != nil {
			logger.Warn("failed to get cluster info for %s: %v", instanceId, err)
			continue
		}
		if clusterInfo.Exists {
			clusters = append(clusters, *clusterInfo)
		}
	}

	logger.Debug("found %d clusters", len(clusters))
	return clusters, nil
}

func (c *Client) CreateCluster(ctx context.Context, instanceId, serviceId, planId string) (string, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: instanceId,
			Labels: map[string]string{
				"cnpg-broker.io/instance-id": instanceId,
			},
			Annotations: map[string]string{
				"cnpg-broker.io/instance-id": instanceId,
			},
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
				"name":      clusterName(instanceId),
				"namespace": instanceId,
				"labels": map[string]any{
					"cnpg-broker.io/instance-id": instanceId,
					"cnpg-broker.io/service-id":  serviceId,
					"cnpg-broker.io/plan-id":     planId,
				},
				"annotations": map[string]any{
					"cnpg-broker.io/instance-id": instanceId,
					"cnpg-broker.io/service-id":  serviceId,
					"cnpg-broker.io/plan-id":     planId,
				},
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
			Name:      fmt.Sprintf("%s-lb-rw", clusterName(instanceId)),
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
				"cnpg.io/cluster":      clusterName(instanceId),
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
					"name":      fmt.Sprintf("%s-pooler", clusterName(instanceId)),
					"namespace": instanceId,
				},
				"spec": map[string]any{
					"cluster": map[string]any{
						"name": clusterName(instanceId),
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
				Name:      fmt.Sprintf("%s-lb-pooler", clusterName(instanceId)),
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
					"cnpg.io/poolerName": fmt.Sprintf("%s-pooler", clusterName(instanceId)),
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
	info := &ClusterInfo{
		Exists:     false,
		InstanceID: instanceId,
		Namespace:  instanceId,
		Name:  clusterName(instanceId),
	}

	cluster, err := c.dynamic.Resource(clusterResource).Namespace(instanceId).Get(ctx, clusterName(instanceId), metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return info, nil
		}
		return nil, err
	}
	info.Exists = true
	info.Labels = cluster.GetLabels()

	// extract annotations
	annotations := cluster.GetAnnotations()
	if serviceId, ok := annotations["cnpg-broker.io/service-id"]; ok {
		info.ServiceID = serviceId
	}
	if planId, ok := annotations["cnpg-broker.io/plan-id"]; ok {
		info.PlanID = planId
	}

	// extract status
	if statusMap, found, err := unstructured.NestedMap(cluster.Object, "status"); found && err == nil {
		if instances, ok := statusMap["instances"].(int64); ok {
			info.TotalInstances = instances
		}
		if ready, ok := statusMap["readyInstances"].(int64); ok {
			info.ReadyInstances = ready
		}
		if phase, ok := statusMap["phase"].(string); ok {
			info.Phase = phase
		}
	}

	// extract specs
	if spec, found, err := unstructured.NestedMap(cluster.Object, "spec"); found && err == nil {
		if instances, ok := spec["instances"].(int64); ok {
			info.Instances = instances
		}
		if resources, found, err := unstructured.NestedMap(spec, "resources"); found && err == nil {
			if requests, found, err := unstructured.NestedMap(resources, "requests"); found && err == nil {
				if cpu, ok := requests["cpu"].(string); ok {
					info.CPU = cpu
				}
				if memory, ok := requests["memory"].(string); ok {
					info.Memory = memory
				}
			}
		}
		if storage, found, err := unstructured.NestedMap(spec, "storage"); found && err == nil {
			if size, ok := storage["size"].(string); ok {
				info.Storage = size
			}
		}
	}

	info.IsFailed = strings.Contains(strings.ToLower(info.Phase), "fail") ||
		strings.Contains(strings.ToLower(info.Phase), "error")
	info.IsReady = info.ReadyInstances == info.TotalInstances && info.TotalInstances > 0 && info.TotalInstances == info.Instances && !info.IsFailed
	info.IsProvisioning = info.Exists && !info.IsReady && !info.IsFailed

	if info.IsFailed {
		info.FailureReason = info.Phase
	}
	return info, nil
}

func (c *Client) DeleteCluster(ctx context.Context, instanceId string) error {
	return c.clientset.CoreV1().Namespaces().Delete(ctx, instanceId, metav1.DeleteOptions{})
}

func (c *Client) UpdateCluster(ctx context.Context, instanceId, planId string, instances int64, cpu, memory, storage string) error {
	cluster, err := c.dynamic.Resource(clusterResource).Namespace(instanceId).Get(ctx, clusterName(instanceId), metav1.GetOptions{})
	if err != nil {
		return err
	}

	// update plan annotation
	annotations := cluster.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["cnpg-broker.io/plan-id"] = planId
	cluster.SetAnnotations(annotations)

	// update specs
	if err := unstructured.SetNestedField(cluster.Object, instances, "spec", "instances"); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(cluster.Object, cpu, "spec", "resources", "requests", "cpu"); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(cluster.Object, cpu, "spec", "resources", "limits", "cpu"); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(cluster.Object, memory, "spec", "resources", "requests", "memory"); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(cluster.Object, memory, "spec", "resources", "limits", "memory"); err != nil {
		return err
	}
	if err := unstructured.SetNestedField(cluster.Object, storage, "spec", "storage", "size"); err != nil {
		return err
	}

	// update existing Cluster
	_, err = c.dynamic.Resource(clusterResource).Namespace(instanceId).Update(ctx, cluster, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	// scale out PVC if necessary
	pvcs, err := c.clientset.CoreV1().PersistentVolumeClaims(clusterName(instanceId)).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, pvc := range pvcs.Items {
		pvc.Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse(storage)
		_, err = c.clientset.CoreV1().PersistentVolumeClaims(clusterName(instanceId)).Update(ctx, &pvc, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) GetCredentials(ctx context.Context, instanceId string) (map[string]string, error) {
	logger.Debug("collecting credentials for instance %s", instanceId)

	secretName := fmt.Sprintf("%s-app", clusterName(instanceId))
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
		"ro_host":     fmt.Sprintf("%s-ro", clusterName(instanceId)),
		"ro_uri":      fmt.Sprintf("postgresql://%s:%s@%s-ro.%s.svc.cluster.local:5432/%s", username, password, clusterName(instanceId), instanceId, database),
		"ro_jdbc_uri": fmt.Sprintf("jdbc:postgresql://%s-ro.%s.svc.cluster.local:5432/%s?password=%s&user=%s", clusterName(instanceId), instanceId, database, password, username),
	}

	logger.Debug("retrieved base credentials for instance %s", instanceId)

	caCertSecretName := fmt.Sprintf("%s-ca", clusterName(instanceId))
	caCertSecret, err := c.clientset.CoreV1().Secrets(instanceId).Get(ctx, caCertSecretName, metav1.GetOptions{})
	if err == nil {
		if caCert, ok := caCertSecret.Data["ca.crt"]; ok {
			credentials["ca_cert"] = string(caCert)
			logger.Debug("collected CA certificate for instance %s", instanceId)
		}
	}

	serverCertSecretName := fmt.Sprintf("%s-server", clusterName(instanceId))
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

	poolerCertSecretName := fmt.Sprintf("%s-pooler", clusterName(instanceId))
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

	lbSvcName := fmt.Sprintf("%s-lb-rw", clusterName(instanceId))
	lbSvc, err := c.clientset.CoreV1().Services(instanceId).Get(ctx, lbSvcName, metav1.GetOptions{})
	if err == nil && len(lbSvc.Status.LoadBalancer.Ingress) > 0 {
		lbHost := lbSvc.Status.LoadBalancer.Ingress[0].IP
		if len(lbHost) == 0 {
			lbHost = lbSvc.Status.LoadBalancer.Ingress[0].Hostname
		}
		credentials["lb_host"] = lbHost
		credentials["lb_uri"] = fmt.Sprintf("postgresql://%s:%s@%s:5432/%s", username, password, lbHost, database)
		credentials["lb_jdbc_uri"] = fmt.Sprintf("jdbc:postgresql://%s:5432/%s?password=%s&user=%s", lbHost, database, username, password)

		poolerSvcName := fmt.Sprintf("%s-lb-pooler", clusterName(instanceId))
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

func (c *Client) GetNamespaceStatus(ctx context.Context, instanceId string) (*NamespaceStatus, error) {
	status := &NamespaceStatus{
		Exists: false,
	}

	ns, err := c.clientset.CoreV1().Namespaces().Get(ctx, instanceId, metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return status, nil
		}
		return nil, err
	}

	status.Exists = true
	status.IsTerminating = ns.Status.Phase == corev1.NamespaceTerminating

	return status, nil
}

func (c *Client) CheckServicesReady(ctx context.Context, instanceId string) (bool, error) {
	lbSvc, err := c.clientset.CoreV1().Services(instanceId).Get(ctx,
		fmt.Sprintf("%s-lb-rw", clusterName(instanceId)), metav1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}

	if len(lbSvc.Status.LoadBalancer.Ingress) == 0 {
		return false, nil
	}

	return true, nil
}
