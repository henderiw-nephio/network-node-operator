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

	srlv1alpha1 "github.com/henderiw-nephio/network-node-operator/apis/srlinux/v1alpha1"
	invv1alpha1 "github.com/nokia/k8s-ipam/apis/inv/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *reconciler) getDeployment(ctx context.Context, cr *invv1alpha1.Node) (*appsv1.Deployment, error) {
	paramRef, err := r.getParamRef(ctx, cr)
	if err != nil {
		return nil, err
	}

	if err := r.checkVariants(ctx, cr, paramRef.GetModel()); err != nil {
		return nil, err
	}

	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.GetName(),
			Namespace: cr.GetNamespace(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: srlv1alpha1.GetSelectorLabels(cr.GetName()),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cr.GetName(),
					Namespace: cr.GetNamespace(),
					Labels:    srlv1alpha1.GetSelectorLabels(cr.GetName()),
				},
				Spec: corev1.PodSpec{
					InitContainers:                []corev1.Container{},
					Containers:                    paramRef.GetContainers(cr.GetName()),
					TerminationGracePeriodSeconds: pointer.Int64(srlv1alpha1.TerminationGracePeriodSeconds),
					NodeSelector:                  map[string]string{},
					Affinity:                      srlv1alpha1.GetAffinity(cr.GetName()),
					Volumes:                       paramRef.GetVolumes(),
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(cr, d, r.scheme); err != nil {
		return nil, err
	}
	return d, nil
}

func (r *reconciler) getParamRef(ctx context.Context, cr *invv1alpha1.Node) (*srlv1alpha1.NodeConfig, error) {
	// a parameterRef needs to be provided e.g. for the image or model that is to be deployed
	if cr.Spec.ParametersRef == nil {
		return nil, fmt.Errorf("cannot deploy pod, no parameterref provided")
	}

	if cr.Spec.ParametersRef.APIVersion != srlv1alpha1.GroupVersion.Identifier() ||
		cr.Spec.ParametersRef.Kind != srlv1alpha1.NodeConfigKind ||
		cr.Spec.ParametersRef.Name == "" {
		return nil, fmt.Errorf("cannot deploy pod, apiVersion -want %s -got %s, kind -want %s -got %s, name must be specified -got %s",
			srlv1alpha1.GroupVersion.Identifier(), cr.Spec.ParametersRef.APIVersion,
			srlv1alpha1.NodeConfigKind, cr.Spec.ParametersRef.Kind,
			cr.Spec.ParametersRef.Name,
		)
	}

	namespace := "default"
	if cr.Spec.ParametersRef.Namespace != "" {
		namespace = cr.Spec.ParametersRef.Namespace
	}

	paramRef := &srlv1alpha1.NodeConfig{}
	if err := r.Get(ctx, types.NamespacedName{Name: cr.Spec.ParametersRef.Name, Namespace: namespace}, paramRef); err != nil {
		return nil, err
	}
	return paramRef, nil
}

func (r *reconciler) checkVariants(ctx context.Context, cr *invv1alpha1.Node, model string) error {
	variants := &corev1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{Name: srlv1alpha1.VariantsCfgMapName, Namespace: cr.GetNamespace()}, variants); err != nil {
		return err
	}
	if _, ok := variants.Data[model]; !ok {
		return fmt.Errorf("cannot deploy pod, variant not provided in the configmap, got: %s", model)
	}
	return nil
}
