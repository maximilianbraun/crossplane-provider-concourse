package clients

import (
	"context"
	"fmt"
	"sync"

	goconcourse "github.com/concourse/concourse/go-concourse/concourse"
	"sigs.k8s.io/controller-runtime/pkg/client"

	providerv1alpha1 "github.com/maximilianbraun/crossplane-provider-concourse/apis/v1alpha1"
)

// Cache is a thread-safe cache of Concourse clients keyed by ProviderConfig identity.
type Cache struct {
	mu      sync.Mutex
	clients map[string]goconcourse.Client
}

// NewCache creates a new client cache.
func NewCache() *Cache {
	return &Cache{
		clients: make(map[string]goconcourse.Client),
	}
}

func cacheKey(pc *providerv1alpha1.ProviderConfig) string {
	return fmt.Sprintf("%s@%s", pc.Name, pc.ResourceVersion)
}

// GetOrBuild returns a cached client or builds a new one.
func (c *Cache) GetOrBuild(ctx context.Context, kube client.Client, pc *providerv1alpha1.ProviderConfig) (goconcourse.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := cacheKey(pc)

	if cl, ok := c.clients[key]; ok {
		return cl, nil
	}

	// Evict stale entries for this PC name (different resourceVersion).
	for k := range c.clients {
		if len(k) > len(pc.Name)+1 && k[:len(pc.Name)+1] == pc.Name+"@" {
			delete(c.clients, k)
		}
	}

	cl, err := NewConcourseClient(ctx, kube, pc)
	if err != nil {
		return nil, err
	}

	c.clients[key] = cl
	return cl, nil
}

// Evict removes all cached clients for a given ProviderConfig name.
func (c *Cache) Evict(pcName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k := range c.clients {
		if len(k) > len(pcName)+1 && k[:len(pcName)+1] == pcName+"@" {
			delete(c.clients, k)
		}
	}
}
