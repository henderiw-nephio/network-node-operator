package srlinux

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

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
	NokiaSROSProvider    = "sros.nokia.com"
	defaultSROSImageName = "ghcr.io/nokia/srlinux:latest"
	defaultSROSVariant   = "ixrd3l"
	scrapliGoSROSKey     = "nokia_sros"

	//
	startupInitialDelay      = 15
	startupFailureThreshold  = 3
	startupPeriodSeconds     = 5
	startupSuccessThreshold  = 1
	startupTimeoutSeconds    = 1
	livenessInitialDelay     = 3
	livenessFailureThreshold = 3
	livenessPeriodSeconds    = 15
	livenessSuccessThreshold = 1
	livenessTimeoutSeconds   = 1
	podAffinityWeight        = 100

	// volumes
	//initialConfigVolMntPath  = "/tmp/initial-config"
	//initialConfigCfgMapName  = "sros-initial-config"
	defaultSecretUserNameKey = "username"
	defaultSecretPasswordKey = "password"
	certificateProfileName   = "k8s-profile"
	//certificateVolName         = "serving-cert"
	//certificateVolMntPath      = "serving-certs"
	//initialConfigVolName       = "initial-config-volume"
	licenseCfgMapName = "licenses.sros.nokia.com"
	licensesVolName   = "license"
	licenseFileName   = "license.txt"
	licenseMntPath    = "/nokia/license/"
	hugePagesVolName  = "hugepages"
	hugePagesMntPath  = "/dev/hugepages"
	banner            = `................................................................
	:                  Welcome to Nokia SROS!                      :
	................................................................
	`
)

var (
	//nolint:gochecknoglobals
	defaultCmd = []string{
		"bin/tini",
	}

	//nolint:gochecknoglobals
	defaultArgs = []string{}

	//nolint:gochecknoglobals
	defaultEnv = []corev1.EnvVar{
		{
			Name:  "SRLINUX",
			Value: "1",
		},
	}

	//nolint:gochecknoglobals
	defaultResourceRequests = map[string]string{
		"cpu":    "2",
		"memory": "8Gi",
	}
	defaultResourceLimits = map[string]string{
		"cpu":           "2",
		"memory":        "8Gi",
		"hugepages-1Gi": "8Gi",
	}
)

// Register registers the node in the NodeRegistry.
func Register(r node.NodeRegistry) {
	r.Register(NokiaSROSProvider, func(c client.Client, s *runtime.Scheme) node.Node {
		return &sros{
			Client: c,
			scheme: s,
		}
	})
}

type sros struct {
	client.Client
	scheme *runtime.Scheme
}

func (r *sros) GetNodeConfig(ctx context.Context, cr *invv1alpha1.Node) (*invv1alpha1.NodeConfig, error) {
	// get nodeConfig via paramRef
	nodeConfig, err := r.getNodeConfig(ctx, cr)
	if err != nil {
		return nil, err
	}

	return nodeConfig, nil
}

func (r *sros) GetNodeModelConfig(ctx context.Context, nc *invv1alpha1.NodeConfig) *corev1.ObjectReference {
	return &corev1.ObjectReference{
		APIVersion: invv1alpha1.NodeKindAPIVersion,
		Kind:       invv1alpha1.NodeModelKind,
		Name:       fmt.Sprintf("%s-%s", NokiaSROSProvider, nc.GetModel(defaultSROSVariant)),
		Namespace:  os.Getenv("POD_NAMESPACE"),
	}
}

func (r *sros) GetInterfaces(ctx context.Context, nc *invv1alpha1.NodeConfig) (*invv1alpha1.NodeModel, error) {
	nm := &invv1alpha1.NodeModel{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s", NokiaSROSProvider, nc.GetModel(defaultSROSVariant)),
		Namespace: os.Getenv("POD_NAMESPACE"),
	}, nm); err != nil {
		return nil, err
	}
	return nm, nil
}

func (r *sros) GetNetworkAttachmentDefinitions(ctx context.Context, cr *invv1alpha1.Node, nc *invv1alpha1.NodeConfig) ([]*nadv1.NetworkAttachmentDefinition, error) {
	// todo check node model and get interfaces from the model
	nads := []*nadv1.NetworkAttachmentDefinition{}
	/*
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
	*/
	return nads, nil
}

func (r *sros) GetPersistentVolumeClaims(ctx context.Context, cr *invv1alpha1.Node, nc *invv1alpha1.NodeConfig) ([]*corev1.PersistentVolumeClaim, error) {
	pvcs := []*corev1.PersistentVolumeClaim{}
	for _, pv := range nc.Spec.PersistentVolumes {
		pvc := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", cr.GetName(), pv.Name),
				Namespace: cr.GetNamespace(),
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{},
				Resources: corev1.ResourceRequirements{
					Requests: pv.Requests,
				},
			},
		}
		pvcs = append(pvcs, pvc)
	}
	return pvcs, nil
}

func (r *sros) GetPodSpec(ctx context.Context, cr *invv1alpha1.Node, nc *invv1alpha1.NodeConfig, nads []*nadv1.NetworkAttachmentDefinition) (*corev1.Pod, error) {
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
			Containers:   getContainers(cr.GetName(), nc),
			NodeSelector: map[string]string{},
			Affinity:     getAffinity(cr.GetNamespace()),
			Volumes:      getVolumes(cr.GetName(), nc),
		},
	}

	hashString := getHash(d.Spec)
	if len(d.GetAnnotations()) == 0 {
		d.ObjectMeta.Annotations = map[string]string{}
	}
	d.ObjectMeta.Annotations[invv1alpha1.RevisionHash] = hashString
	d.ObjectMeta.Annotations[invv1alpha1.NephioWiringKey] = "true"
	if os.Getenv("ENABLE_NAD") == "true" {
		d.ObjectMeta.Annotations[nadv1.NetworkAttachmentAnnot] = string(nadAnnotation)
	}

	if err := ctrl.SetControllerReference(cr, d, r.scheme); err != nil {
		return nil, err
	}
	return d, nil
}

func (r *sros) SetInitialConfig(ctx context.Context, cr *invv1alpha1.Node, ips []corev1.PodIP) error {
	secret := &corev1.Secret{}
	// we assume right now the default secret name is equal to the provider
	// this provider username and password
	if err := r.Get(ctx, types.NamespacedName{Namespace: cr.GetNamespace(), Name: NokiaSROSProvider}, secret); err != nil {
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
		scrapliGoSROSKey,
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

func (r *sros) getNodeConfig(ctx context.Context, cr *invv1alpha1.Node) (*invv1alpha1.NodeConfig, error) {

	if cr.Spec.NodeConfig != nil && cr.Spec.NodeConfig.Name != "" {
		nc := &invv1alpha1.NodeConfig{}
		if err := r.Get(ctx, types.NamespacedName{Name: cr.Spec.NodeConfig.Name, Namespace: os.Getenv("POD_NAMESPACE")}, nc); err != nil {
			return nil, err
		}
		return nc, nil

	}
	// the nodeConfig was not provided, we list all nodeConfigs in the cr namespace
	// we check if there is a nodeconfig with the name equal to the cr name + the provider matches
	// if still not found we look at a nodeconfig with name default that matches the provider
	// if still not found we return an empty nodeConfig, which populates the defaults

	opts := []client.ListOption{
		client.InNamespace(os.Getenv("POD_NAMESPACE")),
	}
	ncl := &invv1alpha1.NodeConfigList{}
	if err := r.List(ctx, ncl, opts...); err != nil {
		return nil, err
	}

	for _, nc := range ncl.Items {
		// if there is a nodeconfig with the exact name of the node -> we return this nodeConfig
		if nc.GetName() == cr.GetName() && cr.Spec.Provider == nc.Spec.Provider {
			return &nc, nil
		}
	}
	for _, nc := range ncl.Items {
		// if there is a nodeconfig with the name default -> we return this nodeConfig
		if nc.GetName() == "default" && cr.Spec.Provider == nc.Spec.Provider {
			return &nc, nil
		}

	}
	// if nothing is found we return an empty nodeconfig
	return &invv1alpha1.NodeConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: os.Getenv("POD_NAMESPACE"),
		},
	}, nil
}

func getContainers(name string, nc *invv1alpha1.NodeConfig) []corev1.Container {
	return []corev1.Container{{
		Name:            name,
		Image:           nc.GetImage(defaultSROSImageName),
		Command:         defaultCmd,
		Args:            defaultArgs,
		Env:             defaultEnv,
		Resources:       nc.GetResourceRequirements(defaultResourceRequests, defaultResourceLimits),
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: &corev1.SecurityContext{
			Privileged: pointer.Bool(true),
			RunAsUser:  pointer.Int64(0),
		},
		TTY:          true,
		Stdin:        true,
		VolumeMounts: getVolumeMounts(nc),
		StartupProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"/opt/nokia/bin/startup_probe",
					},
				},
			},
			InitialDelaySeconds: startupInitialDelay,
			FailureThreshold:    startupFailureThreshold,
			PeriodSeconds:       startupPeriodSeconds,
			SuccessThreshold:    startupSuccessThreshold,
			TimeoutSeconds:      startupTimeoutSeconds,
		},
		LivenessProbe: &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"/opt/nokia/bin/liveness_probe",
					},
				},
			},
			InitialDelaySeconds: livenessInitialDelay,
			FailureThreshold:    livenessFailureThreshold,
			PeriodSeconds:       livenessPeriodSeconds,
			SuccessThreshold:    livenessSuccessThreshold,
			TimeoutSeconds:      livenessTimeoutSeconds,
		},
	}}
}

func getAffinity(topology string) *corev1.Affinity {
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: podAffinityWeight,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{{
								Key:      "topo",
								Operator: "In",
								Values:   []string{topology},
							}},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}
}

func getVolumes(_ string, nc *invv1alpha1.NodeConfig) []corev1.Volume {
	vols := []corev1.Volume{}
	vols = append(vols, getHugePagesVolume())

	for _, pv := range nc.Spec.PersistentVolumes {
		vols = append(vols, getPersistentVolume(pv.Name))
	}

	if nc.Spec.LicenseKey != nil {
		vols = append(vols, getLicenseVolume(nc))
	}
	return vols
}

func getVolumeMounts(nc *invv1alpha1.NodeConfig) []corev1.VolumeMount {
	vms := []corev1.VolumeMount{}
	vms = append(vms, getHugePagesVolumeMount())

	for _, pv := range nc.Spec.PersistentVolumes {
		vms = append(vms, getPersistentVolumeMount(pv.Name, pv.MountPath))
	}

	if nc.Spec.LicenseKey != nil {
		vms = append(vms, getLicenseVolumeMount())
	}

	return vms
}

func getLicenseVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      licensesVolName,
		MountPath: licenseMntPath,
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

func getHugePagesVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      hugePagesVolName,
		MountPath: hugePagesMntPath,
	}
}

func getHugePagesVolume() corev1.Volume {
	return corev1.Volume{
		Name: hugePagesVolName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: "HugePages",
			},
		},
	}
}

func getPersistentVolumeMount(name, mounthPath string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      name,
		MountPath: mounthPath,
	}
}

func getPersistentVolume(name string) corev1.Volume {
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: fmt.Sprintf("pvc-%s", name),
			},
		},
	}
}

func getHash(x any) string {
	b, err := json.Marshal(x)
	if err != nil {
		panic(err)
	}
	hash := sha256.Sum256(b)
	return fmt.Sprintf("%x", hash)
}
