package main

import (
	"bytes"
	"os"
	"path"
	"strings"

	certmangerv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	admissionregistration "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "sigs.k8s.io/cluster-api/util/yaml"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/kyaml/filesys"
)

type resourceKey string

const (
	crdKey        resourceKey = "crds"
	otherKey      resourceKey = "other"
	rbacKey       resourceKey = "rbac"
	deploymentKey resourceKey = "deployment"
	serviceKey    resourceKey = "service"
)

var (
	openshifAnnotations = map[string]string{
		"exclude.release.openshift.io/internal-openshift-hosted":      "true",
		"include.release.openshift.io/self-managed-high-availability": "true",
		"include.release.openshift.io/single-node-developer":          "true",
	}
	techPreviewAnnotation      = "release.openshift.io/feature-gate"
	techPreviewAnnotationValue = "TechPreviewNoUpgrade"
)

func processObjects(objs []unstructured.Unstructured, providerName string) map[resourceKey][]unstructured.Unstructured {
	resourceMap := map[resourceKey][]unstructured.Unstructured{}
	finalObjs := []unstructured.Unstructured{}
	rbacObjs := []unstructured.Unstructured{}
	crdObjs := []unstructured.Unstructured{}
	deploymentObjs := []unstructured.Unstructured{}
	serviceObjs := []unstructured.Unstructured{}

	serviceSecretNames := findWebhookServiceSecretName(objs)

	for _, obj := range objs {
		providerCustomizations(&obj, providerName)
		switch obj.GetKind() {
		case "ClusterRole", "Role", "ClusterRoleBinding", "RoleBinding", "ServiceAccount":
			setOpenShiftAnnotations(obj, false)
			setTechPreviewAnnotation(obj)
			rbacObjs = append(rbacObjs, obj)
		case "MutatingWebhookConfiguration", "ValidatingWebhookConfiguration":
			replaceCertManagerAnnotations(&obj)
			finalObjs = append(finalObjs, obj)
		case "CustomResourceDefinition":
			// Filter out IPAM for metal3
			if strings.Contains(strings.ToLower(obj.GetName()), "ipam") {
				break
			}
			replaceCertManagerAnnotations(&obj)
			removeConversionWebhook(&obj)
			setOpenShiftAnnotations(obj, true)
			setTechPreviewAnnotation(obj)
			crdObjs = append(crdObjs, obj)
		case "Service":
			replaceCertMangerServiceSecret(&obj, serviceSecretNames)
			serviceObjs = append(serviceObjs, obj)
			finalObjs = append(finalObjs, obj)
		case "Deployment":
			customizeDeployments(&obj)
			deploymentObjs = append(deploymentObjs, obj)
			finalObjs = append(finalObjs, obj)
		case "Certificate", "Issuer", "Namespace", "Secret": // skip
		}
	}

	resourceMap[rbacKey] = rbacObjs
	resourceMap[crdKey] = crdObjs
	resourceMap[deploymentKey] = deploymentObjs
	resourceMap[serviceKey] = serviceObjs
	resourceMap[otherKey] = finalObjs

	return resourceMap
}

func setOpenShiftAnnotations(obj unstructured.Unstructured, merge bool) {
	if !merge || len(obj.GetAnnotations()) == 0 {
		obj.SetAnnotations(openshifAnnotations)
	}

	anno := obj.GetAnnotations()
	if anno == nil {
		anno = map[string]string{}
	}

	for k, v := range openshifAnnotations {
		anno[k] = v
	}
	obj.SetAnnotations(anno)
}

func setTechPreviewAnnotation(obj unstructured.Unstructured) {
	anno := obj.GetAnnotations()
	if anno == nil {
		anno = map[string]string{}
	}

	anno[techPreviewAnnotation] = techPreviewAnnotationValue
	obj.SetAnnotations(anno)
}

func findWebhookServiceSecretName(objs []unstructured.Unstructured) map[string]string {
	serviceSecretNames := map[string]string{}
	certSecretNames := map[string]string{}

	secretFromCertNN := func(certNN string) (string, bool) {
		if len(certNN) == 0 {
			return "", false
		}
		certName := strings.Split(certNN, "/")[1]
		secretName, ok := certSecretNames[certName]
		if !ok || secretName == "" {
			return "", false
		}
		return secretName, true
	}
	// find service, then cert, then secret
	// return map[certName] = secretName
	for i, obj := range objs {
		switch obj.GetKind() {
		case "Certificate":
			cert := &certmangerv1.Certificate{}
			if err := scheme.Convert(&objs[i], cert, nil); err != nil {
				panic(err)
			}
			certSecretNames[cert.Name] = cert.Spec.SecretName
		}
	}
	for _, obj := range objs {
		switch obj.GetKind() {
		case "CustomResourceDefinition":
			crd := &apiextensionsv1.CustomResourceDefinition{}
			if err := scheme.Convert(&obj, crd, nil); err != nil {
				panic(err)
			}
			if certNN, ok := crd.Annotations["cert-manager.io/inject-ca-from"]; ok {
				secretName, ok := secretFromCertNN(certNN)
				if !ok {
					panic("can't find secret from cert: " + certNN)
				}
				serviceSecretNames[crd.Spec.Conversion.Webhook.ClientConfig.Service.Name] = secretName
			}

		case "MutatingWebhookConfiguration":
			mwc := &admissionregistration.MutatingWebhookConfiguration{}
			if err := scheme.Convert(&obj, mwc, nil); err != nil {
				panic(err)
			}
			if certNN, ok := mwc.Annotations["cert-manager.io/inject-ca-from"]; ok {
				secretName, ok := secretFromCertNN(certNN)
				if !ok {
					panic("can't find secret from cert: " + certNN)
				}
				serviceSecretNames[mwc.Webhooks[0].ClientConfig.Service.Name] = secretName
			}

		case "ValidatingWebhookConfiguration":
			vwc := &admissionregistration.ValidatingWebhookConfiguration{}
			if err := scheme.Convert(&obj, vwc, nil); err != nil {
				panic(err)
			}
			if certNN, ok := vwc.Annotations["cert-manager.io/inject-ca-from"]; ok {
				secretName, ok := secretFromCertNN(certNN)
				if !ok {
					panic("can't find secret from cert:CustomResourceDefinition " + certNN)
				}
				serviceSecretNames[vwc.Webhooks[0].ClientConfig.Service.Name] = secretName
			}
		}
	}
	return serviceSecretNames
}

func customizeDeployments(obj *unstructured.Unstructured) {
	deployment := &appsv1.Deployment{}
	if err := scheme.Convert(obj, deployment, nil); err != nil {
		panic(err)
	}
	deployment.Spec.Template.Spec.PriorityClassName = "system-cluster-critical"

	for i := range deployment.Spec.Template.Spec.Containers {
		container := &deployment.Spec.Template.Spec.Containers[i]
		// Add resource requests
		container.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("10m"),
			corev1.ResourceMemory: resource.MustParse("50Mi"),
		}
		// Remove any existing resource limits. See: https://github.com/openshift/enhancements/blob/master/CONVENTIONS.md#resources-and-limits
		container.Resources.Limits = corev1.ResourceList{}
		// Remove all image references if they are external, they will be substituted operator later
		if !strings.HasPrefix(container.Image, "registry.ci.openshift.org") {
			container.Image = "to.be/replaced:v99"
		}
		if container.Name == "kube-rbac-proxy" {
			container.Image = "registry.ci.openshift.org/openshift:kube-rbac-proxy"
		}
		noFeatureGates := []string{}
		for _, arg := range container.Args {
			if !strings.HasPrefix(arg, "--feature-gates=") {
				noFeatureGates = append(noFeatureGates, arg)
			}
		}
		if len(noFeatureGates) > 0 {
			container.Args = noFeatureGates
		}
	}

	if err := scheme.Convert(deployment, obj, nil); err != nil {
		panic(err)
	}
}

func replaceCertManagerAnnotations(obj *unstructured.Unstructured) {
	anns := obj.GetAnnotations()
	if anns == nil {
		anns = map[string]string{}
	}
	if _, ok := anns["cert-manager.io/inject-ca-from"]; ok {
		anns["service.beta.openshift.io/inject-cabundle"] = "true"
		delete(anns, "cert-manager.io/inject-ca-from")
		obj.SetAnnotations(anns)
	}
}

func replaceCertMangerServiceSecret(obj *unstructured.Unstructured, serviceSecretNames map[string]string) {
	anns := obj.GetAnnotations()
	if anns == nil {
		anns = map[string]string{}
	}
	if name, ok := serviceSecretNames[obj.GetName()]; ok {
		anns["service.beta.openshift.io/serving-cert-secret-name"] = name
		obj.SetAnnotations(anns)
	}
}

func removeConversionWebhook(obj *unstructured.Unstructured) {
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := scheme.Convert(obj, crd, nil); err != nil {
		panic(err)
	}
	crd.Spec.Conversion = nil
	if err := scheme.Convert(crd, obj, nil); err != nil {
		panic(err)
	}
}

// ensureNewLine makes sure that there is one new line at the end of the file for git
func ensureNewLine(b []byte) []byte {
	return append(bytes.TrimRight(b, "\n"), []byte("\n")...)
}

func writeComponentsToManifests(fileName string, objs []unstructured.Unstructured) error {
	if len(objs) == 0 {
		return nil
	}

	combined, err := utilyaml.FromUnstructured(objs)
	if err != nil {
		return err
	}

	return os.WriteFile(path.Join(manifestsPath, fileName), ensureNewLine(combined), 0600)
}

func fetchAndCompileComponents(url string) ([]byte, error) {
	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())

	fSys := filesys.MakeFsOnDisk()

	m, err := k.Run(fSys, url)
	if err != nil {
		return nil, err
	}

	return m.AsYaml()
}
