package node

import (
	"fmt"
	"sort"
	"strings"
)

type Initializer func() Node

type NodeRegistry interface {
	Register(provider string, i Initializer)
	NewNodeOfProvider(provider string) (Node, error)
}

func NewNodeRegistry() NodeRegistry {
	return &nodeRegistry{
		nodeIndex: map[string]Initializer{},
	}
}

// nodeRegistry implementation for fast lookup
type nodeRegistry struct {
	nodeIndex map[string]Initializer
}

func (r *nodeRegistry) Register(provider string, i Initializer) {
	r.nodeIndex[provider] = i
}

func (r *nodeRegistry) NewNodeOfProvider(provider string) (Node, error) {
	nodeInitializer, ok := r.nodeIndex[provider]
	if !ok {
		return nil, fmt.Errorf("provider %q is not supported. supported providers are %q", provider, strings.Join(r.getRegisteredProviderNodeNames(), ", "))
	}
	// return a new instance of the requested provider node
	return nodeInitializer(), nil
}

func (r *nodeRegistry) getRegisteredProviderNodeNames() []string {
	var result []string
	for k := range r.nodeIndex {
		result = append(result, k)
	}
	// sort and return
	sort.Strings(result)

	return result
}
