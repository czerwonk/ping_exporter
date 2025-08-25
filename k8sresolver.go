package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type K8sResolver struct {
	clientset *kubernetes.Clientset
}

func NewK8sResolver() (Resolver, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}
	r := &K8sResolver{clientset: clientset}
	return r, nil
}

func (r *K8sResolver) LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error) {
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		return nil, errors.New("invalid service name; expected format <service>.<namespace>")
	}

	serviceName := parts[0]
	namespace := parts[1]
	endpoints, err := r.clientset.CoreV1().Endpoints(namespace).Get(ctx, serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get endpoints for service %s in namespace %s: %w", serviceName, namespace, err)
	}

	var ips []net.IPAddr
	for _, subset := range endpoints.Subsets {
		for _, addr := range subset.Addresses {
			_ip := net.ParseIP(addr.IP)
			ip := net.IPAddr{IP: _ip}
			ips = append(ips, ip)
		}
	}

	if len(ips) == 0 {
		return nil, errors.New("no endpoints found for service")
	}

	return ips, nil
}
