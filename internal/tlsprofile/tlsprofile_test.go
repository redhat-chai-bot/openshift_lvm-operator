package tlsprofile

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/openshift/lvm-operator/v4/internal/cluster"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func TestNewOptions(t *testing.T) {
	tests := []struct {
		name             string
		clusterType      cluster.Type
		wantAPIServerGet int
		wantTLSVersion   uint16
		wantOptions      int
	}{
		{
			name:             "OCP uses the API server TLS profile",
			clusterType:      cluster.TypeOCP,
			wantAPIServerGet: 1,
			wantTLSVersion:   tls.VersionTLS13,
			wantOptions:      2,
		},
		{
			name:        "MicroShift only disables HTTP2",
			clusterType: cluster.TypeMicroShift,
			wantOptions: 1,
		},
		{
			name:        "other clusters only disable HTTP2",
			clusterType: cluster.TypeOther,
			wantOptions: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			require.NoError(t, configv1.Install(scheme))

			apiServer := &configv1.APIServer{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Spec: configv1.APIServerSpec{
					TLSSecurityProfile: &configv1.TLSSecurityProfile{Type: configv1.TLSProfileModernType},
				},
			}

			apiServerGets := 0
			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(apiServer).
				WithInterceptorFuncs(interceptor.Funcs{
					Get: func(ctx context.Context, client client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
						apiServerGets++
						return client.Get(ctx, key, obj, opts...)
					},
				}).
				Build()

			options, profile, err := NewOptions(context.Background(), tt.clusterType, k8sClient, log.Log)
			require.NoError(t, err)
			require.Len(t, options, tt.wantOptions)
			require.Equal(t, tt.wantAPIServerGet, apiServerGets)

			tlsConfig := &tls.Config{}
			for _, option := range options {
				option(tlsConfig)
			}

			require.Equal(t, []string{"http/1.1"}, tlsConfig.NextProtos)
			require.Equal(t, tt.wantTLSVersion, tlsConfig.MinVersion)
			if tt.clusterType == cluster.TypeOCP {
				require.Equal(t, configv1.VersionTLS13, profile.MinTLSVersion)
			} else {
				require.Empty(t, profile)
			}
		})
	}
}

func TestNewOptionsReturnsOCPProfileError(t *testing.T) {
	wantErr := errors.New("APIServer lookup failed")
	k8sClient := fake.NewClientBuilder().
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(_ context.Context, _ client.WithWatch, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
				return wantErr
			},
		}).
		Build()

	_, _, err := NewOptions(context.Background(), cluster.TypeOCP, k8sClient, log.Log)
	require.ErrorIs(t, err, wantErr)
}

func TestSetupWatcher(t *testing.T) {
	tests := []struct {
		name        string
		clusterType cluster.Type
		wantCalls   int
		wantErr     bool
	}{
		{
			name:        "OCP registers the watcher",
			clusterType: cluster.TypeOCP,
			wantCalls:   1,
			wantErr:     true,
		},
		{
			name:        "MicroShift skips the watcher",
			clusterType: cluster.TypeMicroShift,
		},
		{
			name:        "other clusters skip the watcher",
			clusterType: cluster.TypeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupCalls := 0
			wantErr := errors.New("watcher setup failed")
			err := SetupWatcher(tt.clusterType, func() error {
				setupCalls++
				return wantErr
			})

			require.Equal(t, tt.wantCalls, setupCalls)
			if tt.wantErr {
				require.ErrorIs(t, err, wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAPIServerCacheByObject(t *testing.T) {
	tests := []struct {
		name        string
		clusterType cluster.Type
		wantObjects int
	}{
		{
			name:        "OCP caches the APIServer",
			clusterType: cluster.TypeOCP,
			wantObjects: 1,
		},
		{
			name:        "MicroShift does not cache the APIServer",
			clusterType: cluster.TypeMicroShift,
		},
		{
			name:        "other clusters do not cache the APIServer",
			clusterType: cluster.TypeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			byObject := APIServerCacheByObject(tt.clusterType)
			require.Len(t, byObject, tt.wantObjects)
			for object := range byObject {
				require.IsType(t, &configv1.APIServer{}, object)
			}
		})
	}
}
