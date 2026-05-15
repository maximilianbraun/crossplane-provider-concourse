package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
)

// BasicAuthCredentials holds credentials for the OAuth2 password-grant flow.
type BasicAuthCredentials struct {
	Username          string                 `json:"username"`
	PasswordSecretRef xpv1.SecretKeySelector `json:"passwordSecretRef"`
}

// TLSConfig configures TLS for the Concourse connection.
type TLSConfig struct {
	// +optional
	CASecretRef *xpv1.SecretKeySelector `json:"caSecretRef,omitempty"`
	// +optional
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
}

// ProviderCredentials defines how to authenticate to Concourse.
type ProviderCredentials struct {
	// +optional
	BasicAuth *BasicAuthCredentials `json:"basicAuth,omitempty"`
	// +optional
	BearerTokenSecretRef *xpv1.SecretKeySelector `json:"bearerTokenSecretRef,omitempty"`
}

// ProviderConfigSpec defines the desired state of a ProviderConfig.
type ProviderConfigSpec struct {
	// URL is the base URL of the Concourse ATC.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^https?://`
	URL string `json:"url"`

	Credentials ProviderCredentials `json:"credentials"`

	// +optional
	TLS *TLSConfig `json:"tls,omitempty"`
}

// ProviderConfigStatus represents the status of a ProviderConfig.
type ProviderConfigStatus struct {
	xpv1.ProviderConfigStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,concourse}
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.spec.url`

// A ProviderConfig configures how this provider connects to Concourse CI.
type ProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderConfigSpec   `json:"spec"`
	Status ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProviderConfigList contains a list of ProviderConfig.
type ProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfig `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster,categories={crossplane,provider,concourse}

// A ProviderConfigUsage indicates that a resource is using a ProviderConfig.
type ProviderConfigUsage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	xpv1.ProviderConfigUsage `json:",inline"`
}

// +kubebuilder:object:root=true

// ProviderConfigUsageList contains a list of ProviderConfigUsage.
type ProviderConfigUsageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfigUsage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ProviderConfig{}, &ProviderConfigList{})
	SchemeBuilder.Register(&ProviderConfigUsage{}, &ProviderConfigUsageList{})
}
