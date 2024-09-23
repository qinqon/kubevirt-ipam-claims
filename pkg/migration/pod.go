package migration

import (
	"context"
	"sort"

	v1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubevirtv1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
)

func EnsureL2MigrationArgs(ctx context.Context, cli client.Client, nse *v1.NetworkSelectionElement, virtLauncherPod *corev1.Pod, vmi *virtv1.VirtualMachineInstance) error {
	if PodCompleted(virtLauncherPod) {
		return nil
	}
	l2MigrationArgs, err := generateL2MigrationArgs(ctx, cli, virtLauncherPod, vmi)
	if err != nil {
		return err
	}
	if l2MigrationArgs != nil {
		cniArgs := map[string]interface{}{
			"ovn.k8s.org/l2-migration": l2MigrationArgs,
		}
		nse.CNIArgs = &cniArgs
		klog.Infof("DELETEME, EnsureL2MigrationArgs, virtLauncherPod: %s, l2MigrationArgs: %+v", virtLauncherPod.Name, l2MigrationArgs)
	}
	return nil
}

func generateL2MigrationArgs(ctx context.Context, cli client.Client, virtLauncherPod *corev1.Pod, vmi *virtv1.VirtualMachineInstance) (*L2MigrationArgs, error) {
	migration, err := findLastMigration(ctx, cli, vmi)
	if err != nil {
		return nil, err
	}
	if migration == nil {
		return nil, nil
	}
	podRole, err := generateL2MigrationPodRole(ctx, cli, virtLauncherPod, migration)
	if err != nil {
		return nil, err
	}
	return &L2MigrationArgs{
		PortName: vmi.Name,
		PodRole:  podRole,
		State:    generateL2MigrationState(migration.Status.MigrationState),
	}, nil
}

func generateL2MigrationPodRole(ctx context.Context, cli client.Client, virtLauncherPod *corev1.Pod, migration *virtv1.VirtualMachineInstanceMigration) (string, error) {
	if migration.Status.MigrationState.SourcePod == virtLauncherPod.Name {
		return "Source", nil
	}
	return "Target", nil
}

func generateL2MigrationState(migrationState *virtv1.VirtualMachineInstanceMigrationState) string {
	if MigrationCompleted(migrationState) {
		return "Completed"
	} else if migrationState.TargetNodeDomainReadyTimestamp != nil {
		return "TargetTrafficReady"
	}
	return "InProgress"
}
func sortVirtLauncherPods(ctx context.Context, cli client.Client, vmiNamespace, vmiName string) ([]*corev1.Pod, error) {
	virtLauncherPods := corev1.PodList{}
	if err := cli.List(ctx, &virtLauncherPods, client.InNamespace(vmiNamespace), client.MatchingLabels{kubevirtv1.VirtualMachineNameLabel: vmiName}); err != nil {
		return nil, err
	}
	if len(virtLauncherPods.Items) == 0 {
		return []*corev1.Pod{}, nil
	}
	activeVirtLauncherPods := []*corev1.Pod{}
	for _, virtLauncherPod := range virtLauncherPods.Items {
		if PodCompleted(&virtLauncherPod) {
			continue
		}
		activeVirtLauncherPods = append(activeVirtLauncherPods, &virtLauncherPod)
	}

	sort.Slice(activeVirtLauncherPods, func(i, j int) bool {
		return activeVirtLauncherPods[i].CreationTimestamp.After(activeVirtLauncherPods[j].CreationTimestamp.Time)
	})
	return activeVirtLauncherPods, nil
}

func PodCompleted(pod *corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed
}
