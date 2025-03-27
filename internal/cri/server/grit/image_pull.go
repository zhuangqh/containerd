package grit

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/containerd/log"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// InterceptPullImage will polling wait for the checkpoint image to be downloaded.
func InterceptPullImage(ctx context.Context, r *runtime.PullImageRequest) error {
	checkpointPath, ok := r.GetSandboxConfig().GetAnnotations()[AnnotationGritCheckpoint]
	if !ok {
		return nil
	}

	sentileFile := path.Join(checkpointPath, CheckpointFileSentinelFile)
	log.G(ctx).Infof("Found restoration mode, waiting for download to complete. sentileFile: %s", sentileFile)
	// Polling wait for the file to come up
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var timeout <-chan time.Time
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.After(time.Until(deadline))
	} else {
		timeout = time.After(10 * time.Minute)
	}

	for {
		select {
		case <-ticker.C:
			if _, err := os.Stat(sentileFile); err == nil {
				log.G(ctx).Infof("File %s is ready", sentileFile)
				return nil
			}
		case <-timeout:
			return fmt.Errorf("timed out waiting for file %s", sentileFile)
		case <-ctx.Done():
			return fmt.Errorf("context canceled while waiting for file %s: %w", sentileFile, ctx.Err())
		}
	}
}
