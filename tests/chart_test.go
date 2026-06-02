package tests

import (
	maps0 "maps"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

const releaseName = "minio-config-cli"

func TestMain(m *testing.M) {
	err := exec.Command("helm", "dependency", "update", "../chart").Run()
	if err != nil {
		panic(err)
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}

func chartPath(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("../chart")
	require.NoError(t, err)
	return p
}

func renderJob(t *testing.T, values, strValues map[string]string) batchv1.Job {
	t.Helper()
	options := &helm.Options{SetValues: values, SetStrValues: strValues}
	output := helm.RenderTemplate(t, options, chartPath(t), releaseName, []string{"templates/job.yaml"})
	var job batchv1.Job
	helm.UnmarshalK8SYaml(t, output, &job)
	return job
}

func envNames(env []corev1.EnvVar) []string {
	out := make([]string, 0, len(env))
	for _, e := range env {
		out = append(out, e.Name)
	}
	return out
}

func mergeStringMaps(maps ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, m := range maps {
		maps0.Copy(out, m)
	}
	return out
}

func TestHelmChartTemplateRequiredValues(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		values map[string]string
	}{
		{
			"MissingURL",
			map[string]string{
				"accessKey": "test-access-key",
				"secretKey": "test-secret-key",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(subT *testing.T) {
			subT.Parallel()

			options := &helm.Options{SetValues: testCase.values}
			_, err := helm.RenderTemplateE(subT, options, chartPath(subT), releaseName, []string{})
			require.Error(subT, err)
		})
	}
}

func TestJobNameHashChangesWithValues(t *testing.T) {
	t.Parallel()

	baseValues := map[string]string{
		"url":       "http://minio:9000",
		"accessKey": "ak",
		"secretKey": "sk",
	}

	baseName := renderJob(t, baseValues, nil).Name

	testCases := []struct {
		name        string
		override    map[string]string
		overrideStr map[string]string
	}{
		{name: "URLChanges", override: map[string]string{"url": "http://other:9000"}},
		{name: "AccessKeyChanges", override: map[string]string{"accessKey": "ak2"}},
		{name: "SecretKeyChanges", override: map[string]string{"secretKey": "sk2"}},
		{name: "OIDCIssuerURLChanges", override: map[string]string{"oidcIssuerUrl": "https://keycloak/realms/x"}},
		{name: "OIDCClientIDChanges", override: map[string]string{"oidcClientId": "minio-client"}},
		{name: "OIDCClientSecretChanges", override: map[string]string{"oidcClientSecret": "oidc-secret"}},
		{name: "OIDCExtraScopesChanges", override: map[string]string{"oidcExtraScopes[0]": "openid"}},
		{name: "GrantTypeChanges", override: map[string]string{"grantType": "password"}},
		{name: "UsernameChanges", override: map[string]string{"username": "alice"}},
		{name: "PasswordChanges", override: map[string]string{"password": "p4ss"}},
		{name: "ExtraEnvVarsChanges", override: map[string]string{
			"extraEnvVars[0].name":  "FOO",
			"extraEnvVars[0].value": "bar",
		}},
		{name: "ConfigChanges", overrideStr: map[string]string{
			"config": "buckets:\n  - name: test-bucket",
		}},
		{name: "ExtraConfigChanges", overrideStr: map[string]string{
			"extraConfig": "users:\n  - name: test-user",
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(subT *testing.T) {
			subT.Parallel()
			name := renderJob(subT, mergeStringMaps(baseValues, tc.override), tc.overrideStr).Name
			require.NotEqual(subT, baseName, name, "Job name should change when %s", tc.name)
		})
	}
}

func TestJobRendersAuthEnvVars(t *testing.T) {
	t.Parallel()

	t.Run("StaticModeRendersEnvVars", func(t *testing.T) {
		t.Parallel()
		job := renderJob(t, map[string]string{
			"url":       "http://minio:9000",
			"accessKey": "ak",
			"secretKey": "sk",
		}, nil)
		require.Len(t, job.Spec.Template.Spec.Containers, 1)
		names := envNames(job.Spec.Template.Spec.Containers[0].Env)
		require.Contains(t, names, "MINIO_ACCESS_KEY")
		require.Contains(t, names, "MINIO_SECRET_KEY")
		require.NotContains(t, names, "OIDC_ISSUER_URL")
		require.NotContains(t, names, "OIDC_CLIENT_ID")
		// Positional args: only the URL.
		require.Equal(t, []string{"import", "http://minio:9000", "--import-file-location=/configs"},
			job.Spec.Template.Spec.Containers[0].Args)
	})

	t.Run("OIDCModeRendersEnvVars", func(t *testing.T) {
		t.Parallel()
		job := renderJob(t, map[string]string{
			"url":              "https://minio.example.com",
			"oidcIssuerUrl":    "https://keycloak.example.com/realms/minio",
			"oidcClientId":     "minio-client",
			"oidcClientSecret": "secret",
		}, nil)
		names := envNames(job.Spec.Template.Spec.Containers[0].Env)
		require.Contains(t, names, "OIDC_ISSUER_URL")
		require.Contains(t, names, "OIDC_CLIENT_ID")
		require.Contains(t, names, "OIDC_CLIENT_SECRET")
		require.NotContains(t, names, "MINIO_ACCESS_KEY")
		require.NotContains(t, names, "MINIO_SECRET_KEY")
	})
}
