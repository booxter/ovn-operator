package ovndbcluster

import (
	"context"
	"fmt"

	infranetworkv1 "github.com/openstack-k8s-operators/infra-operator/apis/network/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/ovn-operator/api/v1beta1"
	ovnv1 "github.com/openstack-k8s-operators/ovn-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Create DNSRecord
func DnsData(
	ctx context.Context,
	helper *helper.Helper,
	svc *corev1.Service,
	ip string,
	instance *ovnv1.OVNDBCluster,
	ovnPod corev1.Pod,
	serviceLabels map[string]string,
) error {
	// ovsdbserver-(nb|sb)-n entry
	dnsHost := infranetworkv1.DNSHost{}
	dnsHost.IP = ip
	dnsHost.Hostnames = []string{svc.ObjectMeta.Annotations[infranetworkv1.AnnotationHostnameKey]}
	// ovsdbserver-(sb|nb) entry
	headlessDnsHostname := ServiceNameSB + "." + instance.Namespace + ".svc"
	if instance.Spec.DBType == v1beta1.NBDBType {
		headlessDnsHostname = ServiceNameNB + "." + instance.Namespace + ".svc"
	}
	dnsHostCname := infranetworkv1.DNSHost{}
	dnsHostCname.IP = ip
	dnsHostCname.Hostnames = append(dnsHostCname.Hostnames, headlessDnsHostname)

	// Create DNSData object
	dnsData := &infranetworkv1.DNSData{
		ObjectMeta: metav1.ObjectMeta{
			Name:      v1beta1.DNSPrefix + "-" + ovnPod.Name,
			Namespace: ovnPod.Namespace,
			Labels:    serviceLabels,
		},
	}
	dnsHosts := []infranetworkv1.DNSHost{dnsHost, dnsHostCname}

	_, err := controllerutil.CreateOrPatch(ctx, helper.GetClient(), dnsData, func() error {
		dnsData.Spec.Hosts = dnsHosts
		dnsData.Spec.DNSDataLabelSelectorValue = "dnsdata"
		err := controllerutil.SetControllerReference(helper.GetBeforeObject(), dnsData, helper.GetScheme())
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		err = fmt.Errorf("Error creating dnsData %s: %w", dnsData.Name, err)
		return err
	}
	return nil
}

func GetDBAddress(instance *ovnv1.OVNDBCluster, svc *corev1.Service) string {
	if svc == nil {
		return ""
	}
	var headlessDnsHostname string
	if instance.Spec.DBType == v1beta1.NBDBType {
		headlessDnsHostname = ServiceNameNB + "." + instance.Namespace + ".svc"
	} else {
		headlessDnsHostname = ServiceNameSB + "." + instance.Namespace + ".svc"
	}
	return fmt.Sprintf("tcp:%s:%d", headlessDnsHostname, svc.Spec.Ports[0].Port)
}
