// Copyright 2026 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/utils/ptr"
)

func TestExtractConfigureTenantRequestOmitsNilResolutionPolicy(t *testing.T) {
	spec := ArchiveMetricsTargetSpec{
		S3Target: &S3MetricsTarget{
			Region:     ptr.To("eu-west-1"),
			BucketName: ptr.To("metrics-bucket"),
		},
	}

	req, err := spec.ExtractConfigureTenantRequest()
	require.NoError(t, err)
	require.NotNil(t, req.S3)
	require.Equal(t, "eu-west-1", *req.S3.Region)
	require.Equal(t, "metrics-bucket", *req.S3.Bucket)
	require.Nil(t, req.RetentionPolicy)
}

func TestExtractConfigureTenantRequestIncludesResolutionPolicy(t *testing.T) {
	spec := ArchiveMetricsTargetSpec{
		S3Target: &S3MetricsTarget{
			Region:     ptr.To("eu-west-1"),
			BucketName: ptr.To("metrics-bucket"),
		},
		ResolutionPolicy: &ResolutionPolicy{
			RawResolution:         ptr.To(int64(1)),
			FiveMinutesResolution: ptr.To(int64(2)),
			OneHourResolution:     ptr.To(int64(3)),
		},
	}

	req, err := spec.ExtractConfigureTenantRequest()
	require.NoError(t, err)
	require.NotNil(t, req.RetentionPolicy)
	require.Equal(t, int64(1), *req.RetentionPolicy.RawResolution)
	require.Equal(t, int64(2), *req.RetentionPolicy.FiveMinutesResolution)
	require.Equal(t, int64(3), *req.RetentionPolicy.OneHourResolution)
}

func TestExtractConfigureTenantRequestRequiresS3Target(t *testing.T) {
	_, err := (&ArchiveMetricsTargetSpec{}).ExtractConfigureTenantRequest()
	require.EqualError(t, err, "archive metrics target does not have a S3Target")
}
