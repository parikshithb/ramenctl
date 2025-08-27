// SPDX-FileCopyrightText: The RamenDR authors
// SPDX-License-Identifier: Apache-2.0

package validate

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/ramendr/ramenctl/pkg/gathering"
	"github.com/ramendr/ramenctl/pkg/report"
)

const (
	deploymentPlural = "deployments"
)

func readDeployment(
	reader gathering.OutputReader,
	name, namespace string,
) (*appsv1.Deployment, error) {
	resource := appsv1.GroupName + "/" + deploymentPlural
	data, err := reader.ReadResource(namespace, resource, name)
	if err != nil {
		return nil, err
	}
	deployment := &appsv1.Deployment{}
	if err := yaml.Unmarshal(data, deployment); err != nil {
		return nil, err
	}
	return deployment, nil
}

func validatedDeploymentCondition(
	condition *appsv1.DeploymentCondition,
	expectedStatus corev1.ConditionStatus,
) report.ValidatedCondition {
	validated := report.ValidatedCondition{Type: string(condition.Type)}

	if condition.Status == expectedStatus {
		validated.State = report.OK
	} else {
		validated.State = report.Problem
		validated.Description = condition.Message
	}

	return validated
}
