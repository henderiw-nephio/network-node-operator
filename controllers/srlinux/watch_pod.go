/*
Copyright 2023 The Nephio Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package srlinux

/*
import (
	"context"

	"github.com/go-logr/logr"
	invv1alpha1 "github.com/nokia/k8s-ipam/apis/inv/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type nodeEventHandler struct {
	client client.Client
	l      logr.Logger
}

// Create enqueues a request for all ip allocation within the ipam
func (e *nodeEventHandler) Create(ctx context.Context, evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	e.add(ctx, evt.Object, q)
}

// Create enqueues a request for all ip allocation within the ipam
func (e *nodeEventHandler) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	e.add(ctx, evt.ObjectOld, q)
	e.add(ctx, evt.ObjectNew, q)
}

// Create enqueues a request for all ip allocation within the ipam
func (e *nodeEventHandler) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	e.add(ctx, evt.Object, q)
}

// Create enqueues a request for all ip allocation within the ipam
func (e *nodeEventHandler) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	e.add(ctx, evt.Object, q)
}

func (e *nodeEventHandler) add(ctx context.Context, obj runtime.Object, queue adder) {
	cr, ok := obj.(*corev1.Pod)
	if !ok {
		return
	}
	e.l = log.FromContext(ctx)
	e.l.Info("event", "kind", obj.GetObjectKind(), "name", cr.GetName())

	if !isOwned(cr) {
		return
	}

	e.l.Info("event requeue network", "name", cr.GetName())
	queue.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: cr.GetNamespace(),
		Name:      cr.GetName()}})

}

func isOwned(o client.Object) bool {
	for _, ownRef := range o.GetOwnerReferences() {
		if ownRef.APIVersion == invv1alpha1.GroupVersion.Identifier() &&
			ownRef.Kind == invv1alpha1.NodeKind {
			return true
		}
	}
	return false
}
*/