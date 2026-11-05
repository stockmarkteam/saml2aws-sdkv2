package commands

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/stretchr/testify/require"
)

func resolveSTSEndpoint(t *testing.T, cfg aws.Config) string {
	t.Helper()
	ep, err := sts.NewDefaultEndpointResolverV2().ResolveEndpoint(context.Background(),
		sts.EndpointParameters{Region: aws.String(cfg.Region)})
	require.NoError(t, err)
	return ep.URI.String()
}

func isolateAWSEnv(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("AWS_CONFIG_FILE", dir+"/config")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", dir+"/credentials")
	for _, k := range []string{"AWS_REGION", "AWS_DEFAULT_REGION", "AWS_PROFILE", "AWS_DEFAULT_PROFILE"} {
		t.Setenv(k, "")
	}
}

func TestLoadAnonymousSTSConfig_NoRegionUsesGlobalEndpoint(t *testing.T) {
	isolateAWSEnv(t)
	cfg, err := loadAnonymousSTSConfig(context.Background(), "")
	require.NoError(t, err)
	require.Equal(t, "https://sts.amazonaws.com", resolveSTSEndpoint(t, cfg))
}

func TestLoadAnonymousSTSConfig_MissingProfileNoRegionUsesGlobalEndpoint(t *testing.T) {
	isolateAWSEnv(t)
	t.Setenv("AWS_PROFILE", "not-created-yet")
	cfg, err := loadAnonymousSTSConfig(context.Background(), "")
	require.NoError(t, err)
	require.Equal(t, "https://sts.amazonaws.com", resolveSTSEndpoint(t, cfg))
}

func TestLoadAnonymousSTSConfig_RegionUsesRegionalEndpoint(t *testing.T) {
	isolateAWSEnv(t)
	cfg, err := loadAnonymousSTSConfig(context.Background(), "ap-northeast-1")
	require.NoError(t, err)
	require.Equal(t, "https://sts.ap-northeast-1.amazonaws.com", resolveSTSEndpoint(t, cfg))
}
