package standalone

import (
	"testing"
)

func TestDefaultResourceLimits(t *testing.T) {
	limits := DefaultResourceLimits()
	if limits.MaxMemory != 256 {
		t.Fatalf("expected MaxMemory 256, got %d", limits.MaxMemory)
	}
	if limits.MaxCPU != 5000 {
		t.Fatalf("expected MaxCPU 5000, got %d", limits.MaxCPU)
	}
	if limits.MaxProcesses != 100 {
		t.Fatalf("expected MaxProcesses 100, got %d", limits.MaxProcesses)
	}
	if limits.MaxOpenFiles != 1024 {
		t.Fatalf("expected MaxOpenFiles 1024, got %d", limits.MaxOpenFiles)
	}
}

func TestApplyUlimit(t *testing.T) {
	limits := DefaultResourceLimits()
	limits.Nice = -5

	existing := []string{"--config", "/etc/some.conf"}
	args := ApplyUlimit(limits, existing)

	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d: %v", len(args), args)
	}
	if args[0] != "-n" {
		t.Fatalf("expected -n, got %s", args[0])
	}
	if args[1] != "-5" {
		t.Fatalf("expected -5, got %s", args[1])
	}
	if args[2] != "--config" {
		t.Fatalf("expected --config, got %s", args[2])
	}
}

func TestApplyUlimit_NoNice(t *testing.T) {
	limits := DefaultResourceLimits()
	limits.Nice = 0

	existing := []string{"--config", "/etc/some.conf"}
	args := ApplyUlimit(limits, existing)

	if len(args) != 2 {
		t.Fatalf("expected 2 args (unchanged), got %d: %v", len(args), args)
	}
}

func TestResourceLimitsValues(t *testing.T) {
	limits := ResourceLimits{
		MaxMemory:    512,
		MaxCPU:       7500,
		MaxProcesses: 200,
		MaxOpenFiles: 2048,
		Nice:         5,
		IOClass:      "idle",
	}
	if limits.MaxMemory != 512 {
		t.Fatal("MaxMemory mismatch")
	}
	if limits.MaxCPU != 7500 {
		t.Fatal("MaxCPU mismatch")
	}
	if limits.MaxProcesses != 200 {
		t.Fatal("MaxProcesses mismatch")
	}
	if limits.MaxOpenFiles != 2048 {
		t.Fatal("MaxOpenFiles mismatch")
	}
	if limits.Nice != 5 {
		t.Fatal("Nice mismatch")
	}
	if limits.IOClass != "idle" {
		t.Fatal("IOClass mismatch")
	}
}
