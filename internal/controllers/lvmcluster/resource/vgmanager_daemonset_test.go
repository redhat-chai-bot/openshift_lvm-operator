package resource

import (
	"fmt"
	"testing"

	lvmv1alpha1 "github.com/openshift/lvm-operator/v4/api/v1alpha1"
	"github.com/openshift/lvm-operator/v4/internal/cluster"
	"github.com/stretchr/testify/assert"
)

func TestVGManagerDaemonSetReceivesClusterType(t *testing.T) {
	for _, clusterType := range []cluster.Type{
		cluster.TypeOCP,
		cluster.TypeMicroShift,
		cluster.TypeOther,
	} {
		t.Run(string(clusterType), func(t *testing.T) {
			ds := templateVGManagerDaemonset(
				&lvmv1alpha1.LVMCluster{},
				clusterType,
				"namespace",
				"image",
				nil,
				[]string{"--zap-log-level=debug"},
			)

			assert.Equal(t, []string{
				"/lvms",
				"vgmanager",
				"--zap-log-level=debug",
				fmt.Sprintf("--cluster-type=%s", clusterType),
			}, ds.Spec.Template.Spec.Containers[0].Command)
		})
	}
}
