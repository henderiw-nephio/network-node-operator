/*
Copyright 2023 Nokia.

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
package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const (
	srlNodeLabelKey         = Group + "/" + "node"
	defaultSRLinuxImageName = "ghcr.io/nokia/srlinux:latest"
	defaultSrlinuxVariant   = "ixrd3l"

	//
	TerminationGracePeriodSeconds = 0
	readinessFile                 = "/etc/opt/srlinux/devices/app_ephemeral.mgmt_server.ready_for_config"
	readinessInitialDelay         = 10
	readinessPeriodSeconds        = 5
	readinessFailureThreshold     = 10
	srlinuxPodAffinityWeight      = 100

	// volumes
	certificateVolName       = "certificate"
	certificateVolMntPath    = "k8s-certs"
	initialConfigVolName     = "initial-config-volume"
	initialConfigVolMntPath  = "/tmp/initial-config"
	initialConfigCfgMapName  = "srlinux-initial-config"
	variantsVolName          = "variants"
	variantsVolMntPath       = "/tmp/topo"
	variantsTemplateTempName = "topo-template.yml"
	VariantsCfgMapName       = "srlinux-variants"
	topomacVolName           = "topomac-script"
	topomacVolMntPath        = "/tmp/topomac"
	topomacCfgMapName        = "srlinux-topomac-script"
	entrypointVolName        = "k8s-entrypoint"
	entrypointVolMntPath     = "/k8s-entrypoint.sh"
	entrypointVolMntSubPath  = "k8s-entrypoint.sh"
	entrypointCfgMapName     = "srlinux-k8s-entrypoint"
	fileMode777              = 0o777
	licenseCfgMapName        = "licenses.srl.nokia.com"
	licensesVolName          = "license"
	licenseFileName          = "license.key"
	licenseMntPath           = "/opt/srlinux/etc/license.key"
	licenseMntSubPath        = "license.key"
)

var (
	//nolint:gochecknoglobals
	defaultCmd = []string{
		"/tini",
		"--",
		"fixuid",
		"-q",
		"/entrypoint.sh",
	}

	//nolint:gochecknoglobals
	defaultArgs = []string{
		"sudo",
		"bash",
		"-c",
		"touch /.dockerenv && /opt/srlinux/bin/sr_linux",
	}

	//nolint:gochecknoglobals
	defaultEnv = []corev1.EnvVar{
		{
			Name:  "SRLINUX",
			Value: "1",
		},
	}

	//nolint:gochecknoglobals
	defaultConstraints = map[string]string{
		"cpu":    "0.5",
		"memory": "1Gi",
	}
)

func (r *NodeConfig) GetModel() string {
	model := defaultSrlinuxVariant
	if r.Spec.Model != nil {
		model = *r.Spec.Model
	}
	return model
}

func (r *NodeConfig) GetImage() string {
	image := defaultSRLinuxImageName
	if r.Spec.Image != nil {
		image = *r.Spec.Image
	}
	return image
}

func (r *NodeConfig) GetCommand() []string {
	return defaultCmd
}

func (r *NodeConfig) GetArgs() []string {
	return defaultArgs
}

func (r *NodeConfig) GetEnv() []corev1.EnvVar {
	return defaultEnv
}

func (r *NodeConfig) GetContainers(name string) []corev1.Container {
	return []corev1.Container{{
		Name:            name,
		Image:           r.GetImage(),
		Command:         r.GetCommand(),
		Args:            r.GetArgs(),
		Env:             r.GetEnv(),
		Resources:       r.GetResourceRequirements(),
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: &corev1.SecurityContext{
			Privileged: pointer.Bool(true),
			RunAsUser:  pointer.Int64(0),
		},
		VolumeMounts: r.GetVolumeMounts(),
		ReadinessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"cat",
						readinessFile,
					},
				},
			},
			InitialDelaySeconds: readinessInitialDelay,
			PeriodSeconds:       readinessPeriodSeconds,
			FailureThreshold:    readinessFailureThreshold,
		},
	}}
}

func GetAffinity(name string) *corev1.Affinity {
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: srlinuxPodAffinityWeight,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{{
								Key:      "topo",
								Operator: "In",
								Values:   []string{name},
							}},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}
}

func (r *NodeConfig) GetResourceRequirements() corev1.ResourceRequirements {
	constraints := defaultConstraints
	if len(r.Spec.Constraints) != 0 {
		// override the default constraints if they exist
		for k := range defaultConstraints {
			if v, ok := r.Spec.Constraints[k]; ok {
				constraints[k] = v
			}
		}
	}
	req := corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{},
	}
	for k, v := range constraints {
		req.Requests[corev1.ResourceName(k)] = resource.MustParse(v)
	}
	return req
}

func (r *NodeConfig) GetVolumes(name string) []corev1.Volume {
	vols := []corev1.Volume{
		{
			Name: variantsVolName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: VariantsCfgMapName,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  r.GetModel(),
							Path: variantsTemplateTempName,
						},
					},
				},
			},
		},
		{
			Name: topomacVolName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: topomacCfgMapName,
					},
				},
			},
		},
		{
			Name: entrypointVolName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: entrypointCfgMapName,
					},
					DefaultMode: pointer.Int32(fileMode777),
				},
			},
		},
		{
			Name: initialConfigVolName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: initialConfigCfgMapName,
					},
				},
			},
		},
		{
			Name: certificateVolName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  name,
					DefaultMode: pointer.Int32(420),
				},
			},
		},
	}
	if r.Spec.LicenseKey != nil {
		vols = append(vols, r.GetLicenseVolume())
	}
	return vols
}

func (r *NodeConfig) GetVolumeMounts() []corev1.VolumeMount {
	vms := []corev1.VolumeMount{
		{
			Name:      variantsVolName,
			MountPath: variantsVolMntPath,
		},
		{
			Name:      topomacVolName,
			MountPath: topomacVolMntPath,
		},
		{
			Name:      entrypointVolName,
			MountPath: entrypointVolMntPath,
			SubPath:   entrypointVolMntSubPath,
		},
		{
			Name:      initialConfigVolName,
			MountPath: initialConfigVolMntPath,
			ReadOnly:  false,
		},
		{
			Name:      certificateVolName,
			MountPath: certificateVolMntPath,
			ReadOnly:  true,
		},
	}

	if r.Spec.LicenseKey != nil {
		vms = append(vms, getLicenseVolumeMount())
	}

	return vms
}

func getLicenseVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      licensesVolName,
		MountPath: licenseMntPath,
		SubPath:   licenseMntSubPath,
	}
}

func (r *NodeConfig) GetLicenseVolume() corev1.Volume {
	return corev1.Volume{
		Name: licensesVolName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: licenseCfgMapName,
				Items: []corev1.KeyToPath{
					{
						Key:  *r.Spec.LicenseKey, // we have check the pointer ref before, so this is safe
						Path: licenseFileName,
					},
				},
			},
		},
	}
}

func GetSelectorLabels(name string) map[string]string {
	return map[string]string{
		srlNodeLabelKey: name,
	}
}
