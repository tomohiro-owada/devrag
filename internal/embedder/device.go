package embedder

import (
	"fmt"
	"os"
	"runtime"
)

type Device int

const (
	CPU Device = iota
	GPU
)

func (d Device) String() string {
	if d == GPU {
		return "GPU"
	}
	return "CPU"
}

// DetectDevice detects the best available device
func DetectDevice(deviceConfig string, fallbackToCPU bool) Device {
	switch deviceConfig {
	case "auto":
		if hasGPU() {
			fmt.Fprintf(os.Stderr, "[INFO] GPU detected, using GPU\n")
			return GPU
		}
		fmt.Fprintf(os.Stderr, "[INFO] No GPU detected, using CPU\n")
		return CPU

	case "gpu":
		if hasGPU() {
			fmt.Fprintf(os.Stderr, "[INFO] Using GPU as requested\n")
			return GPU
		}
		if fallbackToCPU {
			fmt.Fprintf(os.Stderr, "[WARN] GPU requested but unavailable, falling back to CPU\n")
			return CPU
		}
		fmt.Fprintf(os.Stderr, "[ERROR] GPU required but unavailable\n")
		os.Exit(1)

	case "cpu":
		fmt.Fprintf(os.Stderr, "[INFO] Using CPU as requested\n")
		return CPU

	default:
		fmt.Fprintf(os.Stderr, "[WARN] Unknown device '%s', using CPU\n", deviceConfig)
		return CPU
	}

	return CPU
}

// hasGPU checks if GPU is available
func hasGPU() bool {
	// Platform-specific GPU detection
	switch runtime.GOOS {
	case "darwin":
		// macOS: Check for Metal support (M1/M2/M3)
		return runtime.GOARCH == "arm64"
	case "windows", "linux":
		// Check for NVIDIA GPU (simplified)
		// In production, use nvidia-smi or CUDA runtime check
		return false // Default to CPU for now
	}
	return false
}
