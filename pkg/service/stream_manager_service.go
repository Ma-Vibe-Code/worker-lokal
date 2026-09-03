package service

import (
	"bytes"
	"context"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2/log"

	"github.com/Ma-Vibe-Code/worker-lokal/config"
	"github.com/Ma-Vibe-Code/worker-lokal/pkg/model"
)

type cameraRunner struct {
	camera model.Camera
	cancel context.CancelFunc
}

// StreamManagerService coordinates concurrent FFmpeg RTSP stream relays for all active cameras.
type StreamManagerService struct {
	mu         sync.RWMutex
	runners    map[string]*cameraRunner
	rootCtx    context.Context
	rootCancel context.CancelFunc
	wg         sync.WaitGroup
}

// NewStreamManagerService creates an instance of StreamManagerService.
func NewStreamManagerService() *StreamManagerService {
	ctx, cancel := context.WithCancel(context.Background())
	return &StreamManagerService{
		runners:    make(map[string]*cameraRunner),
		rootCtx:    ctx,
		rootCancel: cancel,
	}
}

// UpsertCamera adds a new camera or restarts an existing camera's relay process if configuration changed.
func (sm *StreamManagerService) UpsertCamera(cam model.Camera) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	existing, exists := sm.runners[cam.ID]

	// If camera is marked inactive, stop it if currently running
	if !cam.IsActive {
		if exists {
			log.Infof("[STREAMER] Camera %s (%s) is inactive. Stopping stream...", cam.ID, cam.Name)
			existing.cancel()
			delete(sm.runners, cam.ID)
		}
		return
	}

	// If already running with identical stream endpoints, keep running without interruption
	if exists {
		if existing.camera.SourceURL == cam.SourceURL &&
			existing.camera.TargetURL == cam.TargetURL &&
			existing.camera.IsActive == cam.IsActive {
			existing.camera.Name = cam.Name
			return
		}

		log.Infof("[STREAMER] Camera %s (%s) configuration updated. Restarting stream...", cam.ID, cam.Name)
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

// RemoveCamera halts and deletes an active camera stream runner.
func (sm *StreamManagerService) RemoveCamera(cameraID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if runner, exists := sm.runners[cameraID]; exists {
		log.Infof("[STREAMER] Removing camera %s (%s)...", cameraID, runner.camera.Name)
		runner.cancel()
		delete(sm.runners, cameraID)
	}
}

// ReconcileCameras syncs running streams against the latest target camera list.
func (sm *StreamManagerService) ReconcileCameras(newCameras []model.Camera) {
	newMap := make(map[string]model.Camera)
	for _, cam := range newCameras {
		newMap[cam.ID] = cam
	}

	// Identify streams that no longer exist
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

	// Upsert all cameras
	for _, cam := range newCameras {
		sm.UpsertCamera(cam)
	}

	log.Infof("[STREAMER] Reconciliation completed. Managing %d active stream(s)", sm.ActiveStreamCount())
}

// ActiveStreamCount returns the count of currently running camera streams.
func (sm *StreamManagerService) ActiveStreamCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.runners)
}

// StopAll gracefully terminates all running camera relay subprocesses.
func (sm *StreamManagerService) StopAll() {
	log.Info("[STREAMER] Terminating all camera stream relay processes...")
	sm.rootCancel()

	sm.mu.Lock()
	for id, runner := range sm.runners {
		runner.cancel()
		delete(sm.runners, id)
	}
	sm.mu.Unlock()

	sm.wg.Wait()
	log.Info("[STREAMER] All camera stream runners halted cleanly.")
}

func (sm *StreamManagerService) runCameraLoop(ctx context.Context, cam model.Camera) {
	defer sm.wg.Done()

	ffmpegPath := config.FFMPEG_PATH.GetValueOrDefault("ffmpeg")
	retrySecs := 5
	if strVal := config.RETRY_INTERVAL_SECONDS.GetValueOrDefault("5"); strVal != "" {
		if val, err := strconv.Atoi(strVal); err == nil && val > 0 {
			retrySecs = val
		}
	}

	log.Infof("[STREAMER][%s] Stream worker started (Source: %s -> Target: %s)", cam.ID, cam.SourceURL, cam.TargetURL)

	for {
		select {
		case <-ctx.Done():
			log.Infof("[STREAMER][%s] Stream worker halted via context cancellation", cam.ID)
			return
		default:
		}

		startTime := time.Now()
		log.Infof("[STREAMER][%s] Launching FFmpeg relay subprocess...", cam.ID)

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

		cmd := exec.CommandContext(ctx, ffmpegPath, args...)
		var stderrBuf bytes.Buffer
		cmd.Stderr = &stderrBuf

		err := cmd.Run()
		duration := time.Since(startTime)

		if ctx.Err() != nil {
			log.Infof("[STREAMER][%s] FFmpeg process exited (context canceled)", cam.ID)
			return
		}

		if err != nil {
			log.Warnf("[STREAMER][%s] FFmpeg exited with error after %v: %v | Stderr: %s",
				cam.ID, duration.Round(time.Second), err, stderrBuf.String())
		} else {
			log.Infof("[STREAMER][%s] FFmpeg exited cleanly after %v", cam.ID, duration.Round(time.Second))
		}

		retryDuration := time.Duration(retrySecs) * time.Second
		log.Infof("[STREAMER][%s] Reconnecting stream in %v...", cam.ID, retryDuration)

		select {
		case <-ctx.Done():
			log.Infof("[STREAMER][%s] Stream worker terminated during retry backoff", cam.ID)
			return
		case <-time.After(retryDuration):
		}
	}
}
