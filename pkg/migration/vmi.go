package migration

import (
	"context"
	"sort"

	virtv1 "kubevirt.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func MigrationCompleted(migrationState *virtv1.VirtualMachineInstanceMigrationState) bool {
	if migrationState == nil {
		return true
	}
	return migrationState.Completed || migrationState.Failed
}

func findLastMigration(ctx context.Context, cli client.Client, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachineInstanceMigration, error) {
	migrationList := virtv1.VirtualMachineInstanceMigrationList{}
	if err := cli.List(ctx, &migrationList, client.InNamespace(vmi.Namespace), client.MatchingLabels{virtv1.MigrationSelectorLabel: vmi.Name}); err != nil {
		return nil, err
	}
	if len(migrationList.Items) == 0 {
		return nil, nil
	}
	sort.Slice(migrationList.Items, func(i, j int) bool {
		return migrationList.Items[i].CreationTimestamp.After(migrationList.Items[j].CreationTimestamp.Time)
	})
	return &migrationList.Items[0], nil
}
