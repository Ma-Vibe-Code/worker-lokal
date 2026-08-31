package streamer

import (
	"bytes"
	"context"
	"log"
	"os/exec"
	"sync"
	"time"

	"github.com/Ma-Vibe-Code/worker-lokal/internal/config"
	"github.com/Ma-Vibe-Code/worker-lokal/internal/models"
)

type cameraRunner struct {
	camera models.Camera
	cancel context.CancelFunc
}

// StreamManager coordinates concurrent FFmpeg streaming processes for all active cameras.
type StreamManager struct {
	cfg        *config.Config
	mu         sync.RWMutex
	runners    map[string]*cameraRunner
	rootCtx    context.Context
	rootCancel context.CancelFunc
	wg         sync.WaitGroup
}

// NewStreamManager creates an instance of StreamManager.
func NewStreamManager(cfg *config.Config) *StreamManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &StreamManager{
		cfg:        cfg,
		runners:    make(map[string]*cameraRunner),
		rootCtx:    ctx,
		rootCancel: cancel,
	}
}

// UpsertCamera adds a new camera or updates an existing camera's stream.
func (sm *StreamManager) UpsertCamera(cam models.Camera) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	existing, exists := sm.runners[cam.ID]

	// If camera is deactivated, stop it if currently running
	if !cam.IsActive {
		if exists {
			log.Printf("[STREAMER] Camera %s (%s) is now inactive. Stopping stream...", cam.ID, cam.Name)
			existing.cancel()
			delete(sm.runners, cam.ID)
		}
		return
	}

	// If already running with exact same parameters, keep it running
	if exists {
		if existing.camera.SourceURL == cam.SourceURL &&
			existing.camera.TargetURL == cam.TargetURL &&
			existing.camera.IsActive == cam.IsActive {
			// Configuration identical, no restart needed
			existing.camera.Name = cam.Name
			return
		}

		log.Printf("[STREAMER] Camera %s (%s) config changed. Restarting stream...", cam.ID, cam.Name)
		existing.cancel()
		delete(sm.runners, cam.ID)
	}

	// Start new runner goroutine
	camCtx, camCancel := context.WithCancel(sm.rootCtx)
	sm.runners[cam.ID] = &cameraRunner{
		camera: cam,
		cancel: camCancel,
	}

	sm.wg.Add(1)
	go sm.runCameraLoop(camCtx, cam)
}

// RemoveCamera stops and removes a camera stream.
func (sm *StreamManager) RemoveCamera(cameraID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if runner, exists := sm.runners[cameraID]; exists {
		log.Printf("[STREAMER] Removing camera %s (%s)...", cameraID, runner.camera.Name)
		runner.cancel()
		delete(sm.runners, cameraID)
	}
}

// ReconcileCameras synchronizes the current active streams with the provided camera list.
func (sm *StreamManager) ReconcileCameras(newCameras []models.Camera) {
	newMap := make(map[string]models.Camera)
	for _, cam := range newCameras {
		newMap[cam.ID] = cam
	}

	// Find streams to remove
	sm.mu.RLock()
	var toRemove []string
	for id := range sm.runners {
		if _, exists := newMap[id]; !exists {
			toRemove = append(toRemove, id)
		}
	}
	sm.mu.RUnlock()

	for _, id := range toRemove {
		sm.RemoveCamera(id)
	}

	// Upsert all cameras in new list
	for _, cam := range newCameras {
		sm.UpsertCamera(cam)
	}

	log.Printf("[STREAMER] Reconciliation complete. Currently managing %d active stream(s)", sm.ActiveStreamCount())
}

// ActiveStreamCount returns the number of currently managed camera runners.
func (sm *StreamManager) ActiveStreamCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.runners)
}

// StopAll gracefully shuts down all active camera stream runners.
func (sm *StreamManager) StopAll() {
	log.Println("[STREAMER] Stopping all camera stream runners...")
	sm.rootCancel()

	sm.mu.Lock()
	for id, runner := range sm.runners {
		runner.cancel()
		delete(sm.runners, id)
	}
	sm.mu.Unlock()

	sm.wg.Wait()
	log.Println("[STREAMER] All camera stream runners stopped cleanly.")
}

// runCameraLoop runs the FFmpeg pass-through subprocess with isolated retry loop.
func (sm *StreamManager) runCameraLoop(ctx context.Context, cam models.Camera) {
	defer sm.wg.Done()

	log.Printf("[STREAMER][%s] Stream worker started (Source: %s -> Target: %s)", cam.ID, cam.SourceURL, cam.TargetURL)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[STREAMER][%s] Stream worker terminated by context cancellation", cam.ID)
			return
		default:
		}

		startTime := time.Now()
		log.Printf("[STREAMER][%s] Launching FFmpeg relay subprocess...", cam.ID)

		// FFmpeg pass-through bitstream copy via TCP (-c copy) for minimal CPU/RAM footprint
		// ffmpeg -loglevel warning -rtsp_transport tcp -i <source_url> -c copy -f rtsp -rtsp_transport tcp <target_url>
		args := []string{
			"-hide_banner",
			"-loglevel", "warning",
			"-rtsp_transport", "tcp",
			"-i", cam.SourceURL,
			"-c", "copy",
			"-f", "rtsp",
			"-rtsp_transport", "tcp",
			cam.TargetURL,
		}

		cmd := exec.CommandContext(ctx, sm.cfg.FFmpegPath, args...)

		var stderrBuf bytes.Buffer
		cmd.Stderr = &stderrBuf

		err := cmd.Run()
		duration := time.Since(startTime)

		if ctx.Err() != nil {
			// Normal shutdown
			log.Printf("[STREAMER][%s] FFmpeg process terminated (context canceled)", cam.ID)
			return
		}

		if err != nil {
			log.Printf("[STREAMER][%s] FFmpeg exited with error after %v: %v | Stderr: %s",
				cam.ID, duration.Round(time.Second), err, stderrBuf.String())
		} else {
			log.Printf("[STREAMER][%s] FFmpeg process exited cleanly after %v", cam.ID, duration.Round(time.Second))
		}

		// Error isolation: Wait before retry
		retryDuration := time.Duration(sm.cfg.RetryIntervalSeconds) * time.Second
		log.Printf("[STREAMER][%s] Reconnecting stream in %v...", cam.ID, retryDuration)

		select {
		case <-ctx.Done():
			log.Printf("[STREAMER][%s] Stream worker terminated during retry backoff", cam.ID)
			return
		case <-time.After(retryDuration):
		}
	}
}
