package srlinux

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/henderiw-nephio/network-node-operator/pkg/cert"
	"github.com/henderiw-nephio/network-node-operator/pkg/nad"
	"github.com/henderiw-nephio/network-node-operator/pkg/node"
	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	invv1alpha1 "github.com/nokia/k8s-ipam/apis/inv/v1alpha1"
	"github.com/scrapli/scrapligo/driver/opoptions"
	"github.com/scrapli/scrapligo/driver/options"
	"github.com/scrapli/scrapligo/platform"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NokiaSRLinuxProvider = "srlinux.nokia.com"
	//srlNodeLabelKey         = invv1alpha1.GroupVersion.Group + "/" + "node"
	defaultSRLinuxImageName = "ghcr.io/nokia/srlinux:latest"
	defaultSrlinuxVariant   = "ixrd3l"
	scrapliGoSRLinuxKey     = "nokia_srl"

	//
	terminationGracePeriodSeconds = 0
	readinessFile                 = "/etc/opt/srlinux/devices/app_ephemeral.mgmt_server.ready_for_config"
	readinessInitialDelay         = 10
	readinessPeriodSeconds        = 5
	readinessFailureThreshold     = 10
	srlinuxPodAffinityWeight      = 100

	// volumes
	//initialConfigVolMntPath  = "/tmp/initial-config"
	//initialConfigCfgMapName  = "srlinux-initial-config"
	defaultSecretUserNameKey = "username"
	defaultSecretPasswordKey = "password"
	certificateProfileName   = "k8s-profile"
	//certificateVolName         = "serving-cert"
	//certificateVolMntPath      = "serving-certs"
	//initialConfigVolName       = "initial-config-volume"
	variantsVolName            = "variants"
	variantsVolMntPath         = "/tmp/topo"
	variantsTemplateTempName   = "topo-template.yml"
	variantsCfgMapName         = "srlinux-variants"
	topomacVolName             = "topomac-script"
	topomacVolMntPath          = "/tmp/topomac"
	topomacCfgMapName          = "srlinux-topomac-script"
	k8sEntrypointVolName       = "k8s-entrypoint"
	k8sEntrypointVolMntPath    = "/k8s-entrypoint.sh"
	k8sEntrypointVolMntSubPath = "k8s-entrypoint.sh"
	k8sEntrypointCfgMapName    = "srlinux-k8s-entrypoint"
	fileMode777                = 0o777
	licenseCfgMapName          = "licenses.srl.nokia.com"
	licensesVolName            = "license"
	licenseFileName            = "license.key"
	licenseMntPath             = "/opt/srlinux/etc/license.key"
	licenseMntSubPath          = "license.key"
	banner                     = `................................................................
	:                  Welcome to Nokia SR Linux!                  :
	:              Open Network OS for the NetOps era.             :
	:                                                              :
	:    This is a freely distributed official container image.    :
	:                      Use it - Share it                       :
	:                                                              :
	: Get started: https://learn.srlinux.dev                       :
	: Container:   https://go.srlinux.dev/container-image          :
	: Docs:        https://doc.srlinux.dev/22-11                   :
	: Rel. notes:  https://doc.srlinux.dev/rn22-11-2               :
	: YANG:        https://yang.srlinux.dev/v22.11.2               :
	: Discord:     https://go.srlinux.dev/discord                  :
	: Contact:     https://go.srlinux.dev/contact-sales            :
	................................................................
	`
)

var (
	//nolint:gochecknoglobals
	defaultCmd = []string{
		"/tini",
		"--",
		"fixuid",
		"-q",
		k8sEntrypointVolMntPath,
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

// Register registers the node in the NodeRegistry.
func Register(r node.NodeRegistry) {
	r.Register(NokiaSRLinuxProvider, func(c client.Client, s *runtime.Scheme) node.Node {
		return &srl{
			Client: c,
			scheme: s,
		}
	})
}

type srl struct {
	client.Client
	scheme *runtime.Scheme
}

func (r *srl) GetNodeConfig(ctx context.Context, cr *invv1alpha1.Node) (*invv1alpha1.NodeConfig, error) {
	// get nodeConfig via paramRef
	nodeConfig, err := r.getNodeConfig(ctx, cr)
	if err != nil {
		return nil, err
	}

	// validate if the model returned exists in the variant list
	if err := r.checkVariants(ctx, cr, nodeConfig.GetModel(defaultSrlinuxVariant)); err != nil {
		return nil, err
	}
	return nodeConfig, nil
}

func (r *srl) GetInterfaces(nc *invv1alpha1.NodeConfig) ([]node.Interface, error) {
	x, ok := models[nc.GetModel(defaultSrlinuxVariant)]
	if !ok {
		return nil, fmt.Errorf("model not initialized: got: %s", nc.GetModel(defaultSrlinuxVariant))
	}
	return x, nil
}

func (r *srl) GetNetworkAttachmentDefinitions(ctx context.Context, cr *invv1alpha1.Node, nc *invv1alpha1.NodeConfig) ([]*nadv1.NetworkAttachmentDefinition, error) {
	// todo check node model and get interfaces from the model
	nads := []*nadv1.NetworkAttachmentDefinition{}
	ifNames := []string{"e1-1", "e1-2"}
	for _, ifName := range ifNames {
		b, err := nad.GetNadConfig([]nad.PluginConfigInterface{
			nad.WirePlugin{
				PluginCniType: nad.PluginCniType{
					Type: nad.WirePluginType,
				},
				InterfaceName: ifName,
			},
		})
		if err != nil {
			return nil, err
		}

		n := &nadv1.NetworkAttachmentDefinition{
			TypeMeta: metav1.TypeMeta{
				APIVersion: nadv1.SchemeGroupVersion.Identifier(),
				Kind:       reflect.TypeOf(nadv1.NetworkAttachmentDefinition{}).Name(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: cr.GetNamespace(),
				Name:      strings.Join([]string{cr.GetName(), ifName}, "-"),
			},
			Spec: nadv1.NetworkAttachmentDefinitionSpec{
				Config: string(b),
			},
		}
		if err := ctrl.SetControllerReference(cr, n, r.scheme); err != nil {
			return nil, err
		}
		nads = append(nads, n)
	}
	return nads, nil
}

func (r *srl) GetPodSpec(ctx context.Context, cr *invv1alpha1.Node, nc *invv1alpha1.NodeConfig, nads []*nadv1.NetworkAttachmentDefinition) (*corev1.Pod, error) {
	nadAnnotation, err := nad.GetNadAnnotation(nads)
	if err != nil {
		return nil, err
	}

	d := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.GetName(),
			Namespace: cr.GetNamespace(),
		},
		Spec: corev1.PodSpec{
			//InitContainers:                []corev1.Container{},
			Containers:                    getContainers(cr.GetName(), nc),
			TerminationGracePeriodSeconds: pointer.Int64(terminationGracePeriodSeconds),
			NodeSelector:                  map[string]string{},
			Affinity:                      getAffinity(cr.GetName()),
			Volumes:                       getVolumes(cr.GetName(), nc),
		},
	}

	hashString := getHash(d.Spec)
	if len(d.GetAnnotations()) == 0 {
		d.ObjectMeta.Annotations = map[string]string{}
	}
	d.ObjectMeta.Annotations[invv1alpha1.RevisionHash] = hashString
	d.ObjectMeta.Annotations[nadv1.NetworkAttachmentAnnot] = string(nadAnnotation)

	if err := ctrl.SetControllerReference(cr, d, r.scheme); err != nil {
		return nil, err
	}
	return d, nil
}

func (r *srl) SetInitialConfig(ctx context.Context, cr *invv1alpha1.Node, ips []corev1.PodIP) error {
	secret := &corev1.Secret{}
	// we assume right now the default secret name is equal to the provider
	// this provider username and password
	if err := r.Get(ctx, types.NamespacedName{Namespace: cr.GetNamespace(), Name: NokiaSRLinuxProvider}, secret); err != nil {
		return err
	}

	certSecret := &corev1.Secret{}
	// this is used to provide certificate for the gnmi/gnsi/etc servers on the device
	if err := r.Get(ctx, types.NamespacedName{Namespace: cr.GetNamespace(), Name: cr.GetName()}, certSecret); err != nil {
		return err
	}

	certData, err := cert.GetCertificateData(certSecret, certificateProfileName)
	if err != nil {
		return err
	}

	//fmt.Printf("certData: %v\n", *certData)

	p, err := platform.NewPlatform(
		scrapliGoSRLinuxKey,
		ips[0].IP,
		options.WithAuthNoStrictKey(),
		options.WithAuthUsername(string(secret.Data[defaultSecretUserNameKey])),
		options.WithAuthPassword(string(secret.Data[defaultSecretPasswordKey])),
	)
	if err != nil {
		return err
	}
	d, err := p.GetNetworkDriver()
	if err != nil {
		return err
	}
	err = d.Open()
	if err != nil {
		return err
	}
	defer d.Close()

	commands := []string{
		"enter candidate private\n",
		fmt.Sprintf("set / system tls server-profile %s\n", certData.ProfileName),
		fmt.Sprintf("set / system tls server-profile %s authenticate-client false\n", certData.ProfileName),
		fmt.Sprintf("set / system tls server-profile %s key \"%s\" \n", certData.ProfileName, certData.Key),
		fmt.Sprintf("set / system tls server-profile %s certificate \"%s\" \n", certData.ProfileName, certData.Cert),
		fmt.Sprintf("set / system tls server-profile %s trust-anchor \"%s\" \n", certData.ProfileName, certData.CA),
		"set / system lldp admin state enable\n",
		"set / system gnmi-server admin-state enable\n",
		"set / system gnmi-server rate-limit 65000\n",
		"set / system gnmi-server trace-options [ common request response ]\n",
		"set / system gnmi-server network-instance mgmt admin-state enable\n",
		fmt.Sprintf("set / system gnmi-server network-instance mgmt tls-profile %s \n", certData.ProfileName),
		"set / system gnmi-server network-instance mgmt unix-socket admin-state enable\n",
		"set / system gribi-server admin-state enable\n",
		"set / system gribi-server network-instance mgmt admin-state enable\n",
		fmt.Sprintf("set / system gribi-server network-instance mgmt tls-profile %s \n", certData.ProfileName),
		"set / system json-rpc-server admin-state enable\n",
		"set / system json-rpc-server network-instance mgmt http admin-state enable\n",
		"set / system json-rpc-server network-instance mgmt https admin-state enable\n",
		fmt.Sprintf("set / system json-rpc-server network-instance mgmt https tls-profile %s \n", certData.ProfileName),
		"set / system p4rt-server admin-state enable\n",
		"set / system p4rt-server network-instance mgmt admin-state enable\n",
		fmt.Sprintf("set / system p4rt-server network-instance mgmt tls-profile %s \n", certData.ProfileName),
		fmt.Sprintf("set / system banner login-banner \"%s\" \n", banner),
		"commit save",
	}

	//fmt.Printf("commands:\n%v\n", commands)

	_, err = d.SendCommands(commands, opoptions.WithEager())
	if err != nil {
		return err
	}

	return nil

}

func (r *srl) getNodeConfig(ctx context.Context, cr *invv1alpha1.Node) (*invv1alpha1.NodeConfig, error) {
	if cr.Spec.ParametersRef != nil {
		// for srlinux we expect a specific apiversion and kind
		paramRefSpec := cr.Spec.ParametersRef.DeepCopy()
		if paramRefSpec.APIVersion != invv1alpha1.GroupVersion.Identifier() ||
			paramRefSpec.Kind != invv1alpha1.NodeConfigKind {
			return nil, fmt.Errorf("cannot deploy pod, apiVersion -want %s -got %s, kind -want %s -got %s, name must be specified -got %s",
				invv1alpha1.GroupVersion.Identifier(), paramRefSpec.APIVersion,
				invv1alpha1.NodeConfigKind, paramRefSpec.Kind,
				paramRefSpec.Name,
			)
		}

		// if the parameterRef name exists we expect a specific nodeConfig
		if paramRefSpec.Name != "" {
			nc := &invv1alpha1.NodeConfig{}
			if err := r.Get(ctx, types.NamespacedName{Name: paramRefSpec.Name, Namespace: cr.GetNamespace()}, nc); err != nil {
				return nil, err
			}
			return nc, nil
		}
	}

	opts := []client.ListOption{
		client.InNamespace(cr.GetNamespace()),
	}
	ncl := &invv1alpha1.NodeConfigList{}
	if err := r.List(ctx, ncl, opts...); err != nil {
		return nil, err
	}

	for _, nc := range ncl.Items {
		// if there is a nodeconfig with the exact name of the node -> we return this nodeConfig
		if nc.GetName() == cr.GetName() {
			return &nc, nil
		}
		// if there is a nodeconfig with the name default -> we return this nodeConfig
		if nc.GetName() == "default" {
			return &nc, nil
		}

	}
	// if nothing is found we return an empty nodeconfig
	return &invv1alpha1.NodeConfig{}, nil
}

func (r *srl) checkVariants(ctx context.Context, cr *invv1alpha1.Node, model string) error {
	variants := &corev1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{Name: variantsCfgMapName, Namespace: cr.GetNamespace()}, variants); err != nil {
		return err
	}
	if _, ok := variants.Data[model]; !ok {
		return fmt.Errorf("cannot deploy pod, variant not provided in the configmap, got: %s", model)
	}
	return nil
}

func getContainers(name string, nodeConfig *invv1alpha1.NodeConfig) []corev1.Container {
	return []corev1.Container{{
		Name:            name,
		Image:           nodeConfig.GetImage(defaultSRLinuxImageName),
		Command:         defaultCmd,
		Args:            defaultArgs,
		Env:             defaultEnv,
		Resources:       nodeConfig.GetResourceRequirements(defaultConstraints),
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: &corev1.SecurityContext{
			Privileged: pointer.Bool(true),
			RunAsUser:  pointer.Int64(0),
		},
		VolumeMounts: getVolumeMounts(nodeConfig),
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

func getAffinity(name string) *corev1.Affinity {
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

func getVolumes(name string, nodeConfig *invv1alpha1.NodeConfig) []corev1.Volume {
	vols := []corev1.Volume{
		{
			Name: variantsVolName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: variantsCfgMapName,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  nodeConfig.GetModel(defaultSrlinuxVariant),
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
			Name: k8sEntrypointVolName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: k8sEntrypointCfgMapName,
					},
					DefaultMode: pointer.Int32(fileMode777),
				},
			},
		},
		/*
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
		*/
		/*
			{
				Name: strings.Join([]string{certificateProfileName, certificateVolName}, "-"),
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: name,
						//DefaultMode: pointer.Int32(0755),
					},
				},
			},
		*/
	}
	if nodeConfig.Spec.LicenseKey != nil {
		vols = append(vols, getLicenseVolume(nodeConfig))
	}
	return vols
}

func getVolumeMounts(nodeConfig *invv1alpha1.NodeConfig) []corev1.VolumeMount {
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
			Name:      k8sEntrypointVolName,
			MountPath: k8sEntrypointVolMntPath,
			SubPath:   k8sEntrypointVolMntSubPath,
		},
		/*
			{
				Name:      initialConfigVolName,
				MountPath: initialConfigVolMntPath,
				ReadOnly:  false,
			},
		*/
		/*
			{
				Name:      strings.Join([]string{certificateProfileName, certificateVolName}, "-"),
				MountPath: filepath.Join("tmp", certificateProfileName, certificateVolMntPath),
				ReadOnly:  true,
			},
		*/
	}

	if nodeConfig.Spec.LicenseKey != nil {
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

func getLicenseVolume(nodeConfig *invv1alpha1.NodeConfig) corev1.Volume {
	return corev1.Volume{
		Name: licensesVolName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: licenseCfgMapName,
				Items: []corev1.KeyToPath{
					{
						Key:  *nodeConfig.Spec.LicenseKey, // we have check the pointer ref before, so this is safe
						Path: licenseFileName,
					},
				},
			},
		},
	}
}

/*
func GetSelectorLabels(name string) map[string]string {
	return map[string]string{
		srlNodeLabelKey: name,
	}
}
*/

func getHash(x any) string {
	b, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	hash := sha256.Sum256(b)
	return fmt.Sprintf("%x", hash)
}
