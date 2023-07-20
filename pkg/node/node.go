package node

import (
	"context"

	srlv1alpha1 "github.com/henderiw-nephio/network-node-operator/apis/srlinux/v1alpha1"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	invv1alpha1 "github.com/nokia/k8s-ipam/apis/inv/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// Node is an interface that defines the behavior of a node.
type Node interface {
	GetNodeConfig(ctx context.Context, cr *invv1alpha1.Node) (*srlv1alpha1.NodeConfig, error)
	GetPodSpec(ctx context.Context, cr *invv1alpha1.Node, nc *srlv1alpha1.NodeConfig, nads []*nadv1.NetworkAttachmentDefinition) (*corev1.Pod, error)
	GetNetworkAttachmentDefinitions(ctx context.Context, cr *invv1alpha1.Node, nc *srlv1alpha1.NodeConfig) ([]*nadv1.NetworkAttachmentDefinition, error)
	SetInitialConfig(ctx context.Context, cr *invv1alpha1.Node, ips []corev1.PodIP) error
	GetInterfaces(nc *srlv1alpha1.NodeConfig) ([]Interface, error)
}

type Interface struct {
	Name  string
	Speed string
}
