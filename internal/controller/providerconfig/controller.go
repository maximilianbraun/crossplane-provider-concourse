package providerconfig

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/providerconfig"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	providerv1alpha1 "github.com/maximilianbraun/crossplane-provider-concourse/apis/v1alpha1"
)

func Setup(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named(providerconfig.ControllerName("ProviderConfig")).
		For(&providerv1alpha1.ProviderConfig{}).
		Complete(providerconfig.NewReconciler(mgr, resource.ProviderConfigKinds{
			Config:    providerv1alpha1.SchemeGroupVersion.WithKind("ProviderConfig"),
			Usage:     providerv1alpha1.SchemeGroupVersion.WithKind("ProviderConfigUsage"),
			UsageList: providerv1alpha1.SchemeGroupVersion.WithKind("ProviderConfigUsageList"),
		}))
}
