package logs

import (
	"fmt"
	"os"

	. "github.com/redhat-appstudio/e2e-tests/pkg/utils"
	"github.com/redhat-appstudio/e2e-tests/pkg/utils/common"
	"github.com/redhat-appstudio/e2e-tests/pkg/utils/tekton"

	. "github.com/onsi/ginkgo/v2"
)

func StoreTestLogs(testNamespace, jobName string, cs *common.SuiteController, t *tekton.SuiteController) error {
	wd, _ := os.Getwd()
	artifactDir := GetEnv("ARTIFACT_DIR", fmt.Sprintf("%s/tmp", wd))
	testLogsDir := fmt.Sprintf("%s/%s", artifactDir, testNamespace)

	if err := os.MkdirAll(testLogsDir, os.ModePerm); err != nil {
		return err
	}

	if err := cs.StorePodLogs(testNamespace, jobName, testLogsDir); err != nil {
		GinkgoWriter.Printf("Failed to store pod logs: %s", err)
	}

	if err := t.StorePipelineRuns(testNamespace, cs); err != nil {
		GinkgoWriter.Printf("Failed to store pipelineRun logs: %s", err)
	}

	return nil
}
