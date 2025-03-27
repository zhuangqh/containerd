package grit

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/containerd/log"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// InterceptCreateContainer will try to resume the container log from the checkpoint image.
func InterceptCreateContainer(ctx context.Context, r *runtime.CreateContainerRequest) error {
	sandboxConfig := r.GetSandboxConfig()
	checkpointPath, ok := r.GetSandboxConfig().GetAnnotations()[AnnotationGritCheckpoint]
	if !ok {
		return nil
	}

	var logPath string
	config := r.GetConfig()
	if sandboxConfig.GetLogDirectory() != "" && config.GetLogPath() != "" {
		logPath = filepath.Join(sandboxConfig.GetLogDirectory(), config.GetLogPath())
	} else {
		return nil
	}

	savedLogPath := path.Join(checkpointPath, config.GetMetadata().GetName(), CheckpointFileContainerLog)
	if _, err := os.Stat(savedLogPath); err == nil {
		log.G(ctx).Infof("Resume container %q log from %q", config.GetMetadata().GetName(), savedLogPath)
		srcFile, err := os.Open(savedLogPath)
		if err != nil {
			return fmt.Errorf("failed to open source log file %q: %w", savedLogPath, err)
		}
		defer srcFile.Close()

		destFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0640)
		if err != nil {
			return fmt.Errorf("failed to open destination log file %q: %w", logPath, err)
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, srcFile); err != nil {
			return fmt.Errorf("failed to copy log file from %q to %q: %w", savedLogPath, logPath, err)
		}
	} else {
		log.G(ctx).Warnf("Saved log file %q does not exist, skipping log resume", savedLogPath)
	}

	return nil
}
