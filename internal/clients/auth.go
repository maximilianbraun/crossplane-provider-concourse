package clients

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	goconcourse "github.com/concourse/concourse/go-concourse/concourse"

	providerv1alpha1 "github.com/maximilianbraun/crossplane-provider-concourse/apis/v1alpha1"
)

const (
	oauthClientID     = "fly"
	oauthClientSecret = "Zmx5"
	tokenEndpointPath = "/sky/issuer/token"
)

// NewConcourseClient builds a go-concourse client from a ProviderConfig.
func NewConcourseClient(ctx context.Context, kube client.Client, pc *providerv1alpha1.ProviderConfig) (goconcourse.Client, error) {
	httpClient, err := buildHTTPClient(ctx, kube, pc)
	if err != nil {
		return nil, fmt.Errorf("building HTTP client: %w", err)
	}

	return goconcourse.NewClient(pc.Spec.URL, httpClient, false), nil
}

func buildHTTPClient(ctx context.Context, kube client.Client, pc *providerv1alpha1.ProviderConfig) (*http.Client, error) {
	tlsConfig, err := buildTLSConfig(ctx, kube, pc)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}

	creds := pc.Spec.Credentials

	switch {
	case creds.BasicAuth != nil:
		password, err := readSecretValue(ctx, kube, creds.BasicAuth.PasswordSecretRef)
		if err != nil {
			return nil, fmt.Errorf("reading password secret: %w", err)
		}

		tokenURL := pc.Spec.URL + tokenEndpointPath
		oauthConfig := &oauth2.Config{
			ClientID:     oauthClientID,
			ClientSecret: oauthClientSecret,
			Endpoint: oauth2.Endpoint{
				TokenURL: tokenURL,
			},
		}

		baseClient := &http.Client{Transport: transport}
		oauthCtx := context.WithValue(ctx, oauth2.HTTPClient, baseClient)

		token, err := oauthConfig.PasswordCredentialsToken(oauthCtx, creds.BasicAuth.Username, password)
		if err != nil {
			return nil, fmt.Errorf("obtaining OAuth2 token: %w", err)
		}

		tokenSource := oauthConfig.TokenSource(oauthCtx, token)
		return oauth2.NewClient(oauthCtx, oauth2.ReuseTokenSource(token, tokenSource)), nil

	case creds.BearerTokenSecretRef != nil:
		token, err := readSecretValue(ctx, kube, *creds.BearerTokenSecretRef)
		if err != nil {
			return nil, fmt.Errorf("reading bearer token secret: %w", err)
		}

		tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: token,
			TokenType:   "Bearer",
		})
		baseClient := &http.Client{Transport: transport}
		oauthCtx := context.WithValue(ctx, oauth2.HTTPClient, baseClient)
		return oauth2.NewClient(oauthCtx, tokenSource), nil

	default:
		return nil, fmt.Errorf("no credentials configured in ProviderConfig %q", pc.Name)
	}
}

func buildTLSConfig(ctx context.Context, kube client.Client, pc *providerv1alpha1.ProviderConfig) (*tls.Config, error) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}

	if pc.Spec.TLS == nil {
		return cfg, nil
	}

	cfg.InsecureSkipVerify = pc.Spec.TLS.InsecureSkipVerify

	if pc.Spec.TLS.CASecretRef != nil {
		caPEM, err := readSecretValue(ctx, kube, *pc.Spec.TLS.CASecretRef)
		if err != nil {
			return nil, fmt.Errorf("reading CA secret: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(caPEM)) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		cfg.RootCAs = pool
	}

	return cfg, nil
}

func readSecretValue(ctx context.Context, kube client.Client, ref xpv1.SecretKeySelector) (string, error) {
	secret := &corev1.Secret{}
	nn := types.NamespacedName{
		Namespace: ref.SecretReference.Namespace,
		Name:      ref.SecretReference.Name,
	}
	if err := kube.Get(ctx, nn, secret); err != nil {
		return "", fmt.Errorf("getting secret %s: %w", nn, err)
	}
	val, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret %s", ref.Key, nn)
	}
	return string(val), nil
}
