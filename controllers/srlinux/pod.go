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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
)

/*
func (r *reconciler) getDeployment(ctx context.Context, cr *invv1alpha1.Node) (*appsv1.Deployment, error) {
	// get nodeConfig via paramRef
	nodeConfig, err := r.getNodeConfig(ctx, cr)
	if err != nil {
		return nil, err
	}

	if err := r.checkVariants(ctx, cr, nodeConfig.GetModel()); err != nil {
		return nil, err
	}

	d := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.Identifier(),
			Kind:       reflect.TypeOf(appsv1.Deployment{}).Name(),
		},
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
					Containers:                    nodeConfig.GetContainers(cr.GetName()),
					TerminationGracePeriodSeconds: pointer.Int64(srlv1alpha1.TerminationGracePeriodSeconds),
					NodeSelector:                  map[string]string{},
					Affinity:                      srlv1alpha1.GetAffinity(cr.GetName()),
					Volumes:                       nodeConfig.GetVolumes(),
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(cr, d, r.scheme); err != nil {
		return nil, err
	}
	return d, nil
}
*/

func (r *reconciler) getPodSpec(ctx context.Context, cr *invv1alpha1.Node) (*corev1.Pod, error) {
	// get nodeConfig via paramRef
	nodeConfig, err := r.getNodeConfig(ctx, cr)
	if err != nil {
		return nil, err
	}

	if err := r.checkVariants(ctx, cr, nodeConfig.GetModel()); err != nil {
		return nil, err
	}

	d := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.GetName(),
			Namespace: cr.GetNamespace(),
		},
		Spec: corev1.PodSpec{
			//InitContainers:                []corev1.Container{},
			Containers:                    nodeConfig.GetContainers(cr.GetName()),
			TerminationGracePeriodSeconds: pointer.Int64(srlv1alpha1.TerminationGracePeriodSeconds),
			NodeSelector:                  map[string]string{},
			Affinity:                      srlv1alpha1.GetAffinity(cr.GetName()),
			Volumes:                       nodeConfig.GetVolumes(cr.GetName()),
		},
	}

	if err := ctrl.SetControllerReference(cr, d, r.scheme); err != nil {
		return nil, err
	}
	return d, nil
}

func (r *reconciler) getNodeConfig(ctx context.Context, cr *invv1alpha1.Node) (*srlv1alpha1.NodeConfig, error) {
	// a parameterRef needs to be provided e.g. for the image or model that is to be deployed
	paramRefSpec := &corev1.ObjectReference{
		APIVersion: srlv1alpha1.GroupVersion.Identifier(),
		Kind:       srlv1alpha1.NodeConfigKind,
		Name:       cr.GetName(),
		Namespace:  cr.GetNamespace(),
	}
	if cr.Spec.ParametersRef != nil {
		paramRefSpec = cr.Spec.ParametersRef.DeepCopy()
	}

	if paramRefSpec.APIVersion != srlv1alpha1.GroupVersion.Identifier() ||
		paramRefSpec.Kind != srlv1alpha1.NodeConfigKind ||
		paramRefSpec.Name == "" {
		return nil, fmt.Errorf("cannot deploy pod, apiVersion -want %s -got %s, kind -want %s -got %s, name must be specified -got %s",
			srlv1alpha1.GroupVersion.Identifier(), paramRefSpec.APIVersion,
			srlv1alpha1.NodeConfigKind, paramRefSpec.Kind,
			paramRefSpec.Name,
		)
	}

	nc := &srlv1alpha1.NodeConfig{}
	if err := r.Get(ctx, types.NamespacedName{Name: paramRefSpec.Name, Namespace: paramRefSpec.Namespace}, nc); err != nil {
		return nil, err
	}
	return nc, nil
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

func getPodStatus(pod *corev1.Pod) (string, bool) {
	if len(pod.Status.ContainerStatuses) == 0 {
		return "pod conditions empty", false
	}
	if !pod.Status.ContainerStatuses[0].Ready {
		return "pod not ready empty", false
	}
	return "", true
}
