package node

import (
	"context"

	invv1alpha1 "github.com/nokia/k8s-ipam/apis/inv/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Node is an interface that defines the behavior of a node.
type Node interface {
	Init(c client.Client, scheme *runtime.Scheme)
	GetPodSpec(ctx context.Context, cr *invv1alpha1.Node) (*corev1.Pod, error)
	SetInitialConfig(ctx context.Context, cr *invv1alpha1.Node, ips []corev1.PodIP) error
}
