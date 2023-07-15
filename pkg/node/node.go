package node

import (
	"context"

	invv1alpha1 "github.com/nokia/k8s-ipam/apis/inv/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// Node is an interface that defines the behavior of a node.
type Node interface {
	GetPodSpec(ctx context.Context, cr *invv1alpha1.Node) (*corev1.Pod, error)
	SetInitialConfig(ctx context.Context, cr *invv1alpha1.Node, ips []corev1.PodIP) error
}
