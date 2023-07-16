/*
Copyright 2022 Nokia.

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

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	srlv1alpha1 "github.com/henderiw-nephio/network-node-operator/apis/srlinux/v1alpha1"
	"github.com/henderiw-nephio/network-node-operator/controllers"
	"github.com/henderiw-nephio/network-node-operator/controllers/ctrlconfig"
	"github.com/henderiw-nephio/network-node-operator/pkg/node"
	"github.com/nephio-project/nephio/controllers/pkg/resource"
	invv1alpha1 "github.com/nokia/k8s-ipam/apis/inv/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func init() {
	controllers.Register("srlinux", &reconciler{})
}

const (
	finalizer        = "srlinux.nokia.com/finalizer"
	nokiaSRLProvider = "srlinux.nokia.com"
	// errors
	errGetCr        = "cannot get cr"
	errUpdateStatus = "cannot update status"
)

// SetupWithManager sets up the controller with the Manager.
func (r *reconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, c interface{}) (map[schema.GroupVersionKind]chan event.GenericEvent, error) {
	cfg, ok := c.(*ctrlconfig.ControllerConfig)
	if !ok {
		return nil, fmt.Errorf("cannot initialize, expecting controllerConfig, got: %s", reflect.TypeOf(c).Name())
	}

	if err := invv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}
	if err := srlv1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}

	r.Client = mgr.GetClient()
	r.finalizer = resource.NewAPIFinalizer(mgr.GetClient(), finalizer)
	r.scheme = mgr.GetScheme()
	r.nodeRegistry = cfg.Noderegistry

	return nil, ctrl.NewControllerManagedBy(mgr).
		Named("SrlinuxNodeController").
		For(&invv1alpha1.Node{}).
		Owns(&corev1.Pod{}).
		Complete(r)

}

// reconciler reconciles a srlinux node object
type reconciler struct {
	client.Client
	scheme       *runtime.Scheme
	finalizer    *resource.APIFinalizer
	nodeRegistry node.NodeRegistry

	l logr.Logger
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.l = log.FromContext(ctx)
	r.l.Info("reconcile", "req", req)

	cr := &invv1alpha1.Node{}
	if err := r.Get(ctx, req.NamespacedName, cr); err != nil {
		// if the resource no longer exists the reconcile loop is done
		if resource.IgnoreNotFound(err) != nil {
			r.l.Error(err, errGetCr)
			return ctrl.Result{}, errors.Wrap(resource.IgnoreNotFound(err), errGetCr)
		}
		return ctrl.Result{}, nil
	}
	cr = cr.DeepCopy()

	if resource.WasDeleted(cr) {
		if err := r.finalizer.RemoveFinalizer(ctx, cr); err != nil {
			r.l.Error(err, "cannot remove finalizer")
			cr.SetConditions(srlv1alpha1.Failed(err.Error()))
			return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}
		r.l.Info("Successfully deleted resource")
		return ctrl.Result{}, nil
	}

	node, err := r.nodeRegistry.NewNodeOfProvider(cr.Spec.Provider, r.Client, r.scheme)
	if err != nil {
		cr.SetConditions(srlv1alpha1.Failed(err.Error()))
		return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	newpod, err := node.GetPodSpec(ctx, cr)
	if err != nil {
		cr.SetConditions(srlv1alpha1.Failed(err.Error()))
		return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	var create bool
	existingPod := &corev1.Pod{}
	if err := r.Get(ctx, req.NamespacedName, existingPod); err != nil {
		if resource.IgnoreNotFound(err) != nil {
			// an error occurred
			cr.SetConditions(srlv1alpha1.Failed(err.Error()))
			return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}
		// pod does not exist -> indicate to create it
		create = true
	} else {
		r.l.Info("pod exists",
			"oldHash", existingPod.GetAnnotations()[srlv1alpha1.RevisionHash],
			"newHash", newpod.GetAnnotations()[srlv1alpha1.RevisionHash],
		)
		if newpod.GetAnnotations()[srlv1alpha1.RevisionHash] != existingPod.GetAnnotations()[srlv1alpha1.RevisionHash] {
			// pod spec changed, since pods are immutable we delete and create the pod
			r.l.Info("pod spec changed")
			if err := r.Delete(ctx, existingPod); err != nil {
				cr.SetConditions(srlv1alpha1.Failed(err.Error()))
				return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
			}
			create = true
		}
	}

	if create {
		if err := r.Create(ctx, newpod); err != nil {
			cr.SetConditions(srlv1alpha1.Failed(err.Error()))
			return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
		}
	}

	// at this stage the pod should exist
	pod := &corev1.Pod{}
	if err := r.Get(ctx, req.NamespacedName, pod); err != nil {
		cr.SetConditions(srlv1alpha1.Failed(err.Error()))
		return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	podIPs, msg, ready := getPodStatus(pod)
	if !ready {
		cr.SetConditions(srlv1alpha1.NotReady(msg))
		return ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	r.l.Info("pod ips", "ips", podIPs)
	if err := node.SetInitialConfig(ctx, cr, podIPs); err != nil {
		r.l.Error(err, "cannot set initial config")
		cr.SetConditions(srlv1alpha1.Failed(err.Error()))
		return ctrl.Result{Requeue: true}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
	}

	r.l.Info("ready", "req", req)
	cr.SetConditions(srlv1alpha1.Ready())
	return ctrl.Result{}, errors.Wrap(r.Status().Update(ctx, cr), errUpdateStatus)
}
