package vgmanager

import (
	"testing"

	"github.com/openshift/lvm-operator/v4/internal/cluster"
	"github.com/stretchr/testify/assert"
)

func TestParseClusterType(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    cluster.Type
		wantErr bool
	}{
		{name: "OpenShift", value: string(cluster.TypeOCP), want: cluster.TypeOCP},
		{name: "MicroShift", value: string(cluster.TypeMicroShift), want: cluster.TypeMicroShift},
		{name: "other Kubernetes", value: string(cluster.TypeOther), want: cluster.TypeOther},
		{name: "missing", wantErr: true},
		{name: "unknown", value: "unknown", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseClusterType(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
