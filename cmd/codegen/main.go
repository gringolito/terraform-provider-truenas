// cmd/codegen generates the typed TrueNAS client from a method-registry snapshot.
//
// Usage:
//
//	Generate typed Go files from the snapshot:
//	  go run ./cmd/codegen [--snapshot api/registry.json] [--namespaces ...] [--out internal/truenas/]
//
//	Refresh the snapshot from a live TrueNAS box:
//	  go run ./cmd/codegen --refresh --host <host> --api-key <key> [--snapshot api/registry.json] [--namespaces ...]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/gringolito/terraform-provider-truenas/internal/codegen"
)

var defaultNamespaces = []string{
	"user",
	"group",
	"pool.dataset",
	"sharing.nfs",
	"sharing.smb",
	"pool",
}

func main() {
	// Defaults mirror the provider's env-var names so the same shell environment
	// works for both 'terraform apply' and 'make refresh-snapshot'.
	insecureDefault := false
	if v := os.Getenv("TRUENAS_INSECURE_SKIP_VERIFY"); v != "" {
		if parsed, err := strconv.ParseBool(v); err == nil {
			insecureDefault = parsed
		}
	}

	var (
		snapshot   = flag.String("snapshot", "api/registry.json", "path to registry JSON snapshot")
		nsFlag     = flag.String("namespaces", strings.Join(defaultNamespaces, ","), "comma-separated namespace allowlist")
		out        = flag.String("out", "internal/truenas/", "output directory for generated Go files")
		refresh    = flag.Bool("refresh", false, "connect to live TrueNAS, write snapshot, then exit")
		host       = flag.String("host", os.Getenv("TRUENAS_HOST"), "TrueNAS host (also TRUENAS_HOST)")
		apiKey     = flag.String("api-key", os.Getenv("TRUENAS_API_KEY"), "TrueNAS API key (also TRUENAS_API_KEY)")
		skipVerify = flag.Bool("insecure", insecureDefault, "skip TLS certificate verification (also TRUENAS_INSECURE_SKIP_VERIFY)")
	)
	flag.Parse()

	namespaces := parseNamespaces(*nsFlag)

	if *refresh {
		if err := runRefresh(*host, *apiKey, *skipVerify, *snapshot, namespaces); err != nil {
			log.Fatalf("refresh: %v", err)
		}
		return
	}

	if err := runGenerate(*snapshot, namespaces, *out); err != nil {
		log.Fatalf("generate: %v", err)
	}
}

func parseNamespaces(s string) []string {
	var ns []string
	for n := range strings.SplitSeq(s, ",") {
		n = strings.TrimSpace(n)
		if n != "" {
			ns = append(ns, n)
		}
	}
	return ns
}

// runRefresh connects to TrueNAS, calls core.get_methods, filters by namespace,
// and writes the snapshot to disk.
func runRefresh(host, apiKey string, skipVerify bool, snapshotPath string, namespaces []string) error {
	if host == "" {
		return fmt.Errorf("--host is required with --refresh")
	}
	if apiKey == "" {
		return fmt.Errorf("--api-key is required with --refresh")
	}

	log.Printf("connecting to %s …", host)
	c, err := client.NewWebSocketClient(host, apiKey, "", "", skipVerify)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	log.Println("calling core.get_methods …")
	raw, err := c.Call(ctx, "core.get_methods", []any{})
	if err != nil {
		return fmt.Errorf("core.get_methods: %w", err)
	}

	// Unmarshal to map so we can filter by namespace.
	var full map[string]json.RawMessage
	if err := json.Unmarshal(raw, &full); err != nil {
		return fmt.Errorf("unmarshal registry: %w", err)
	}

	log.Printf("received %d methods; filtering to %v …", len(full), namespaces)

	allowed := make(map[string]bool, len(namespaces))
	for _, ns := range namespaces {
		allowed[ns] = true
	}
	filtered := make(map[string]json.RawMessage)
	for method, def := range full {
		ns := methodNamespace(method)
		if allowed[ns] {
			filtered[method] = def
		}
	}

	out, err := json.MarshalIndent(filtered, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(snapshotPath, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", snapshotPath, err)
	}
	log.Printf("wrote %d methods to %s", len(filtered), snapshotPath)
	return nil
}

// runGenerate reads the snapshot and writes typed Go source files to outDir.
func runGenerate(snapshotPath string, namespaces []string, outDir string) error {
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		return fmt.Errorf("read snapshot: %w", err)
	}
	reg, err := codegen.ParseRegistry(data)
	if err != nil {
		return fmt.Errorf("parse registry: %w", err)
	}

	log.Printf("generating typed client for %v → %s", namespaces, outDir)
	if err := codegen.Generate(reg, namespaces, outDir); err != nil {
		return fmt.Errorf("generate: %w", err)
	}
	log.Println("done")
	return nil
}

// methodNamespace returns everything before the last dot.
func methodNamespace(method string) string {
	idx := strings.LastIndex(method, ".")
	if idx < 0 {
		return method
	}
	return method[:idx]
}
