// SPDX-FileCopyrightText: The RamenDR authors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"fmt"
	"path/filepath"

	"github.com/nirs/kubectl-gather/pkg/gather"
	"github.com/ramendr/ramen/e2e/types"

	"github.com/ramendr/ramenctl/pkg/gathering"
	"github.com/ramendr/ramenctl/pkg/ramen"
	"github.com/ramendr/ramenctl/pkg/s3"
)

// Backend performs validation with real clusters.
type Backend struct{}

var _ Validation = &Backend{}

// Validate the environment. Must be called once before calling other functions.
func (b Backend) Validate(ctx Context) error {
	if err := detectDistro(ctx); err != nil {
		return err
	}
	if err := validateClusterset(ctx); err != nil {
		return err
	}
	return nil
}

// ApplicationNamespaces inspects the application DRPC and returns the application namespaces on the
// hub and managed clusters.
func (b Backend) ApplicationNamespaces(
	ctx Context,
	drpcName, drpcNamespace string,
) ([]string, error) {
	drpc, err := ramen.GetDRPC(ctx, drpcName, drpcNamespace)
	if err != nil {
		return nil, err
	}
	return ramen.ApplicationNamespaces(drpc), nil
}

func (b Backend) Gather(
	ctx Context,
	clusters []*types.Cluster,
	options gathering.Options,
) <-chan gathering.Result {
	return gathering.Namespaces(ctx, clusters, options)
}

func (b Backend) GatherS3(
	ctx Context,
	drpcName, drpcNamespace, outputDir string,
) error {
	hub := ctx.Env().Hub
	hubClusterDir := filepath.Join(outputDir, hub.Name)
	reader := gather.NewOutputReader(hubClusterDir)

	ramenS3Profiles, err := ramen.GetS3Profiles(reader, ctx.Config().Namespaces.RamenHubNamespace)
	if err != nil {
		return fmt.Errorf("failed to get S3 profiles from cluster %q: %w", hub.Name, err)
	}

	prefix, err := ramen.ApplicationS3Prefix(reader, drpcName, drpcNamespace)
	if err != nil {
		return err
	}

	for _, profile := range ramenS3Profiles {
		s3Profile := s3.Profile(profile)
		client, err := s3.NewClient(ctx.Context(), s3Profile, ctx.Logger())
		if err != nil {
			return fmt.Errorf("failed to create S3 client for profile %q: %w",
				s3Profile.Name, err)
		}

		if err := client.DownloadObjects(ctx.Context(), prefix, outputDir); err != nil {
			return fmt.Errorf("failed to download objects from profile %q: %w",
				s3Profile.Name, err)
		}
	}

	return nil
}
