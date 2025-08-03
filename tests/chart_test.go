package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
)

func TestMain(m *testing.M) {
	err := exec.Command("helm", "dependency", "update", "../chart").Run()
	if err != nil {
		panic(err)
	}
	exitCode := m.Run()
	os.Exit(exitCode)
}

func TestHelmChartTemplateRequiredValues(t *testing.T) {
	t.Parallel()

	helmChartPath, err := filepath.Abs("../chart")
	require.NoError(t, err)

	releaseName := "minio-config-cli"

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
		{
			"MissingAccessKey",
			map[string]string{
				"url":       "http://minio:9000",
				"secretKey": "test-secret-key",
			},
		},
		{
			"MissingSecretKey",
			map[string]string{
				"url":       "http://minio:9000",
				"accessKey": "test-access-key",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(subT *testing.T) {
			subT.Parallel()

			options := &helm.Options{SetValues: testCase.values}
			_, err := helm.RenderTemplateE(subT, options, helmChartPath, releaseName, []string{})
			require.Error(subT, err)
		})
	}
}

func TestJobNameHashChangesWithConfig(t *testing.T) {
	t.Parallel()

	helmChartPath, err := filepath.Abs("../chart")
	require.NoError(t, err)

	releaseName := "minio-config-cli"

	baseValues := map[string]string{
		"url":       "http://minio:9000",
		"accessKey": "test-access-key",
		"secretKey": "test-secret-key",
	}

	// Render template with initial config
	options1 := &helm.Options{
		SetValues: baseValues,
		SetStrValues: map[string]string{
			"config": `buckets:
  - name: test-bucket`,
		},
	}
	output1 := helm.RenderTemplate(t, options1, helmChartPath, releaseName, []string{"templates/job.yaml"})

	var job1 batchv1.Job
	helm.UnmarshalK8SYaml(t, output1, &job1)

	// Render template with different config
	options2 := &helm.Options{
		SetValues: baseValues,
		SetStrValues: map[string]string{
			"config": `buckets:
  - name: different-bucket`,
		},
	}
	output2 := helm.RenderTemplate(t, options2, helmChartPath, releaseName, []string{"templates/job.yaml"})

	var job2 batchv1.Job
	helm.UnmarshalK8SYaml(t, output2, &job2)

	// Job names should be different when config changes
	require.NotEqual(t, job1.Name, job2.Name, "Job names should be different when config changes")

	// Test extraConfig changes
	options3 := &helm.Options{
		SetValues: baseValues,
		SetStrValues: map[string]string{
			"config": `buckets:
  - name: test-bucket`,
			"extraConfig": `users:
  - name: test-user`,
		},
	}
	output3 := helm.RenderTemplate(t, options3, helmChartPath, releaseName, []string{"templates/job.yaml"})

	var job3 batchv1.Job
	helm.UnmarshalK8SYaml(t, output3, &job3)

	// Job names should be different when extraConfig is added
	require.NotEqual(t, job1.Name, job3.Name, "Job names should be different when extraConfig changes")
	require.NotEqual(t, job2.Name, job3.Name, "Job names should be different when extraConfig changes")
}
