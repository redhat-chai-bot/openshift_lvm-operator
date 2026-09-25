/*
Copyright © 2026 Red Hat, Inc.

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

package tlsprofile

import (
	"context"
	"crypto/tls"

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	ctrlRuntimeCommon "github.com/openshift/controller-runtime-common/pkg/tls"
	"github.com/openshift/lvm-operator/v4/internal/cluster"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewOptions returns TLS options for controller-runtime servers. All cluster
// types disable HTTP/2. Only OCP clusters additionally use the centrally
// configured API server TLS profile.
func NewOptions(ctx context.Context, clusterType cluster.Type, k8sClient client.Client, logger logr.Logger) ([]func(*tls.Config), configv1.TLSProfileSpec, error) {
	tlsOpts := []func(*tls.Config){
		func(c *tls.Config) {
			c.NextProtos = []string{"http/1.1"}
		},
	}

	if clusterType != cluster.TypeOCP {
		return tlsOpts, configv1.TLSProfileSpec{}, nil
	}

	tlsProfile, err := ctrlRuntimeCommon.FetchAPIServerTLSProfile(ctx, k8sClient)
	if err != nil {
		return nil, configv1.TLSProfileSpec{}, err
	}

	tlsConfig, unsupportedCiphers := ctrlRuntimeCommon.NewTLSConfigFromProfile(tlsProfile)
	if len(unsupportedCiphers) > 0 {
		logger.Info("some ciphers from TLS profile are not supported", "unsupportedCiphers", unsupportedCiphers)
	}

	return append(tlsOpts, tlsConfig), tlsProfile, nil
}

// SetupWatcher registers the API server TLS profile watcher on OCP clusters.
func SetupWatcher(clusterType cluster.Type, setup func() error) error {
	if clusterType != cluster.TypeOCP {
		return nil
	}

	return setup()
}

// APIServerCacheByObject returns the cache configuration needed by the API
// server TLS profile watcher. Non-OCP clusters must not create an informer for
// an API that they do not serve.
func APIServerCacheByObject(clusterType cluster.Type) map[client.Object]cache.ByObject {
	if clusterType != cluster.TypeOCP {
		return nil
	}

	return map[client.Object]cache.ByObject{
		&configv1.APIServer{}: {},
	}
}
