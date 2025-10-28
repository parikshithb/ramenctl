// SPDX-FileCopyrightText: The RamenDR authors
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"

	ramenapi "github.com/ramendr/ramen/api/v1alpha1"
	e2etypes "github.com/ramendr/ramen/e2e/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ramendr/ramenctl/pkg/console"
	"github.com/ramendr/ramenctl/pkg/core"
	"github.com/ramendr/ramenctl/pkg/gathering"
	"github.com/ramendr/ramenctl/pkg/ramen"
	"github.com/ramendr/ramenctl/pkg/report"
	"github.com/ramendr/ramenctl/pkg/time"
)

func (c *Command) Application(drpcName, drpcNamespace string) error {
	c.report.Application = &report.Application{
		Name:      drpcName,
		Namespace: drpcNamespace,
	}
	if !c.validateConfig() {
		return c.failed()
	}
	if !c.validateApplication(drpcName, drpcNamespace) {
		return c.failed()
	}
	c.passed()
	return nil
}

func (c *Command) validateApplication(drpcName, drpcNamespace string) bool {
	console.Step("Validate application")
	c.startStep("validate application")

	namespaces, ok := c.inspectApplication(drpcName, drpcNamespace)
	if !ok {
		return c.finishStep()
	}

	c.report.Namespaces = namespaces

	options := gathering.Options{
		Namespaces: namespaces,
		OutputDir:  c.dataDir(),
	}
	if !c.gatherNamespaces(options) {
		return c.finishStep()
	}

	if !c.validateGatheredApplicationData(drpcName, drpcNamespace) {
		return c.finishStep()
	}

	c.finishStep()
	return true
}

func (c *Command) inspectApplication(drpcName, drpcNamespace string) ([]string, bool) {
	start := time.Now()
	step := &report.Step{Name: "inspect application"}
	c.Logger().Infof("Step %q started", step.Name)

	namespaces, err := c.backend.ApplicationNamespaces(c, drpcName, drpcNamespace)
	if err != nil {
		step.Duration = time.Since(start).Seconds()
		if errors.Is(err, context.Canceled) {
			console.Error("Canceled %s", step.Name)
			step.Status = report.Canceled
		} else {
			console.Error("Failed to %s", step.Name)
			step.Status = report.Failed
		}
		c.Logger().Errorf("Step %q %s: %s", c.current.Name, step.Status, err)
		c.current.AddStep(step)

		return nil, false
	}

	step.Duration = time.Since(start).Seconds()
	step.Status = report.Passed
	c.current.AddStep(step)

	console.Pass("Inspected application")
	c.Logger().Infof("Step %q passed", step.Name)

	// For consistent gather order and report.
	slices.Sort(namespaces)

	return namespaces, true
}

func (c *Command) validateGatheredApplicationData(drpcName, drpcNamespace string) bool {
	log := c.Logger()

	start := time.Now()
	step := &report.Step{Name: "validate data"}
	defer func() {
		step.Duration = time.Since(start).Seconds()
		c.current.AddStep(step)
	}()

	s := &report.ApplicationStatus{}
	c.report.ApplicationStatus = s

	drpc, err := c.validateApplicationHub(&s.Hub, drpcName, drpcNamespace)
	if err != nil {
		step.Status = report.Failed
		msg := "Failed to validate hub"
		console.Error(msg)
		log.Errorf("%s: %s", msg, err)
		return false
	}

	if err := c.validateApplicationPrimaryCluster(&s.PrimaryCluster, drpc); err != nil {
		step.Status = report.Failed
		msg := "Failed to validate primary cluster"
		console.Error(msg)
		log.Errorf("%s: %s", msg, err)
		return false
	}

	if err := c.validateApplicationSecondaryCluster(&s.SecondaryCluster, drpc); err != nil {
		step.Status = report.Failed
		msg := "Failed to validate secondary cluster"
		console.Error(msg)
		log.Errorf("%s: %s", msg, err)
		return false
	}

	if c.report.Summary.HasIssues() {
		step.Status = report.Failed
		msg := "Issues found during validation"
		console.Error(msg)
		log.Errorf("%s: %s", msg, c.report.Summary)
		return false
	}

	step.Status = report.Passed
	console.Pass("Application validated")
	return true
}

func (c *Command) validateApplicationHub(
	s *report.ApplicationStatusHub,
	drpcName, drpcNamespace string,
) (*ramenapi.DRPlacementControl, error) {
	log := c.Logger()
	reader := c.outputReader(c.Env().Hub.Name)
	drpc, err := ramen.ReadDRPC(reader, drpcName, drpcNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to read drpc: %w", err)
	}
	log.Debugf("Read drpc \"%s/%s\"", drpc.Namespace, drpc.Name)
	c.validateApplicationDRPC(&s.DRPC, drpc)
	return drpc, nil
}

func (c *Command) validateApplicationPrimaryCluster(
	s *report.ApplicationStatusCluster,
	drpc *ramenapi.DRPlacementControl,
) error {
	cluster, err := ramen.PrimaryCluster(c, drpc)
	if err != nil {
		return fmt.Errorf("failed to find primary cluster: %w", err)
	}
	s.Name = cluster.Name
	return c.validateApplicationVRG(&s.VRG, cluster, drpc, ramenapi.PrimaryState)
}

func (c *Command) validateApplicationSecondaryCluster(
	s *report.ApplicationStatusCluster,
	drpc *ramenapi.DRPlacementControl,
) error {
	cluster, err := ramen.SecondaryCluster(c, drpc)
	if err != nil {
		return fmt.Errorf("failed to find secondary cluster: %w", err)
	}
	s.Name = cluster.Name
	return c.validateApplicationVRG(&s.VRG, cluster, drpc, ramenapi.SecondaryState)
}

func (c *Command) validateApplicationDRPC(
	s *report.DRPCSummary,
	drpc *ramenapi.DRPlacementControl,
) {
	s.Name = drpc.Name
	s.Namespace = drpc.Namespace
	s.Deleted = c.validatedDeleted(drpc)
	s.DRPolicy = drpc.Spec.DRPolicyRef.Name
	s.Action = c.validatedDRPCAction(string(drpc.Spec.Action))
	s.Phase = c.validatedDRPCPhase(drpc)
	s.Progression = c.validatedDRPCProgression(drpc)
	s.Conditions = c.validatedConditions(drpc, drpc.Status.Conditions)
}

func (c *Command) validateApplicationVRG(
	s *report.VRGSummary,
	cluster *e2etypes.Cluster,
	drpc *ramenapi.DRPlacementControl,
	stableState ramenapi.State,
) error {
	log := c.Logger()
	reader := c.outputReader(cluster.Name)
	vrgName := drpc.Name
	vrgNamespace := ramen.VRGNamespace(drpc)

	vrg, err := ramen.ReadVRG(reader, vrgName, vrgNamespace)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to read vrg from cluster %q: %w", cluster.Name, err)
		}

		log.Debugf("vrg \"%s/%s\" missing in cluster %q", vrgNamespace, vrgName, cluster.Name)
		s.Name = vrgName
		s.Namespace = vrgNamespace
		s.Deleted = c.validatedDeleted(nil)
		return nil
	}

	log.Debugf("Read vrg \"%s/%s\" from cluster %q", vrgNamespace, vrgName, cluster.Name)
	s.Name = vrgName
	s.Namespace = vrgNamespace
	s.Deleted = c.validatedDeleted(vrg)
	s.Conditions = c.validatedVRGConditions(vrg)
	s.ProtectedPVCs = c.validatedProtectedPVCs(cluster, vrg)
	s.PVCGroups = c.pvcGroups(vrg)
	s.State = c.validatedVRGState(vrg, stableState)

	return nil
}

func (c *Command) validatedDRPCPhase(drpc *ramenapi.DRPlacementControl) report.ValidatedString {
	validated := report.ValidatedString{Value: string(drpc.Status.Phase)}

	// We expect stable phase as ok, and anything else as an error. An application is not expected
	// to be in unstable phase (e.g. FailingOver) for a long time. The stable phase depends on the
	// action.

	stablePhase, err := ramen.StablePhase(drpc.Spec.Action)
	if err != nil {
		validated.State = report.Problem
		validated.Description = err.Error()
	} else {
		if drpc.Status.Phase != stablePhase {
			validated.State = report.Problem
			validated.Description = fmt.Sprintf("Waiting for stable phase %q", stablePhase)
		} else {
			validated.State = report.OK
		}
	}

	c.report.Summary.Add(&validated)
	return validated
}

func (c *Command) validatedDRPCProgression(
	drpc *ramenapi.DRPlacementControl,
) report.ValidatedString {
	validated := report.ValidatedString{Value: string(drpc.Status.Progression)}

	// We expect a stable progression (Completed). An application should not be in unstable state
	// for long time, so it we see unstable progression it requires investigation.
	if drpc.Status.Progression != ramenapi.ProgressionCompleted {
		validated.State = report.Problem
		validated.Description = fmt.Sprintf(
			"Waiting for progression %q",
			ramenapi.ProgressionCompleted,
		)
	} else {
		validated.State = report.OK
	}

	c.report.Summary.Add(&validated)
	return validated
}

func (c *Command) validatedVRGState(
	vrg *ramenapi.VolumeReplicationGroup,
	stableState ramenapi.State,
) report.ValidatedString {
	validated := report.ValidatedString{Value: string(vrg.Status.State)}

	// We expect the stable state. An application should not be in unstable state for long time, so
	// it we see unstable state it requires investigation.
	if vrg.Status.State != stableState {
		validated.State = report.Problem
		validated.Description = fmt.Sprintf("Waiting to become %q", stableState)
	} else {
		validated.State = report.OK
	}

	c.report.Summary.Add(&validated)
	return validated
}

func (c *Command) validatedProtectedPVCPhase(
	pvc *corev1.PersistentVolumeClaim,
) report.ValidatedString {
	validated := report.ValidatedString{Value: string(pvc.Status.Phase)}

	// Protected PVC must be bound; anything else seen for long time requires investigation.
	if pvc.Status.Phase != corev1.ClaimBound {
		validated.State = report.Problem
		validated.Description = fmt.Sprintf("PVC is not %q", corev1.ClaimBound)
	} else {
		validated.State = report.OK
	}

	c.report.Summary.Add(&validated)
	return validated
}

func (c *Command) validatedDRPCAction(action string) report.ValidatedString {
	validated := report.ValidatedString{Value: action}
	if slices.Contains(ramen.Actions, action) {
		validated.State = report.OK
	} else {
		validated.State = report.Problem
		validated.Description = fmt.Sprintf("Unknown action %q", action)
	}
	c.report.Summary.Add(&validated)
	return validated
}

func (c *Command) validatedProtectedPVCs(
	cluster *e2etypes.Cluster,
	vrg *ramenapi.VolumeReplicationGroup,
) []report.ProtectedPVCSummary {
	log := c.Logger()

	// Protected PVCs becomes stale on a secondary cluster:
	// https://github.com/RamenDR/ramenctl/issues/286.
	if vrg.Status.State == ramenapi.SecondaryState {
		log.Debugf(
			"Skipping protected pvcs on cluster %q for vrg state %q",
			cluster.Name, vrg.Status.State,
		)
		return nil
	}

	reader := c.outputReader(cluster.Name)
	var protectedPVCs []report.ProtectedPVCSummary

	for i := range vrg.Status.ProtectedPVCs {
		ppvc := &vrg.Status.ProtectedPVCs[i]
		ps := report.ProtectedPVCSummary{
			Name:        ppvc.Name,
			Namespace:   ppvc.Namespace,
			Replication: c.protectedPVCReplication(ppvc),
			Conditions:  c.validatedProtectedPVCConditions(vrg, ppvc),
		}

		if pvc, err := core.ReadPVC(reader, ppvc.Name, ppvc.Namespace); err != nil {
			log.Warnf("failed to read pvc \"%s/%s\" from cluster %q: %s",
				ppvc.Namespace, ppvc.Name, cluster.Name, err)
			ps.Deleted = c.validatedDeleted(nil)
		} else {
			log.Debugf("Read pvc \"%s/%s\" from cluster %q", pvc.Namespace, pvc.Name, cluster.Name)
			ps.Deleted = c.validatedDeleted(pvc)
			ps.Phase = c.validatedProtectedPVCPhase(pvc)
		}

		protectedPVCs = append(protectedPVCs, ps)
	}

	return protectedPVCs
}

func (c *Command) validatedVRGConditions(
	vrg *ramenapi.VolumeReplicationGroup,
) []report.ValidatedCondition {
	var conditions []report.ValidatedCondition
	for i := range vrg.Status.Conditions {
		condition := &vrg.Status.Conditions[i]
		// On the secondary cluster most conditions are unused.
		if condition.Reason == ramen.VRGConditionReasonUnused {
			continue
		}
		// DataProtected behaves differently for volrep and volsync. Since a workload can have both
		// volsync protected pvcs and volrep protected pvcs we seem to have now way to validate this
		// condition.
		if condition.Type == ramen.VRGConditionTypeDataProtected {
			continue
		}
		validated := validatedCondition(vrg, condition, metav1.ConditionTrue)
		c.report.Summary.Add(&validated)
		conditions = append(conditions, validated)
	}
	return conditions
}

func (c *Command) protectedPVCReplication(ppvc *ramenapi.ProtectedPVC) report.ReplicationType {
	// TODO: report external replication.
	if ppvc.ProtectedByVolSync {
		return report.Volsync
	}
	return report.Volrep
}

func (c *Command) validatedProtectedPVCConditions(
	vrg *ramenapi.VolumeReplicationGroup,
	ppvc *ramenapi.ProtectedPVC,
) []report.ValidatedCondition {
	log := c.Logger()

	var conditions []report.ValidatedCondition
	for i := range ppvc.Conditions {
		condition := &ppvc.Conditions[i]

		// DataProtected exists only with volrep and has confusing and unhelpful semantics. Status
		// is False in the stable state and True during some part of Relocate phase.
		if condition.Type == ramen.VRGConditionTypeDataProtected {
			continue
		}

		// Volsync PVsRestored condition is always stale on the primary after failover or
		// relocate, but the application is fine.
		if condition.Type == ramen.VRGConditionTypeVolSyncPVsRestored &&
			condition.ObservedGeneration != vrg.Generation {
			log.Debugf(
				"Skipping stale protected PVC condition: observed generation %d does not match vrg generation: %+v",
				condition.ObservedGeneration,
				vrg.Generation,
				condition,
			)
			continue
		}

		validated := validatedCondition(vrg, condition, metav1.ConditionTrue)
		c.report.Summary.Add(&validated)
		conditions = append(conditions, validated)
	}
	return conditions
}

func (c *Command) pvcGroups(vrg *ramenapi.VolumeReplicationGroup) []report.PVCGroupsSummary {
	if len(vrg.Status.PVCGroups) == 0 {
		return nil
	}

	groups := make([]report.PVCGroupsSummary, 0, len(vrg.Status.PVCGroups))
	for _, group := range vrg.Status.PVCGroups {
		if len(group.Grouped) > 0 {
			groups = append(groups, report.PVCGroupsSummary{Grouped: group.Grouped})
		}
	}
	return groups
}
