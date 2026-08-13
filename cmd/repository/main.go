package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	runnerpackage "github.com/nodima-studio/nodima-sdk/packagekit"
	runnerv1 "github.com/nodima-studio/nodima-sdk/runner/v1"
)

const releaseBase = "https://github.com/nodima-studio/node.repository/releases/download"

type catalog struct {
	FormatVersion string           `json:"formatVersion"`
	Packages      []catalogPackage `json:"packages"`
}
type catalogPackage struct {
	ID       string           `json:"id"`
	Summary  string           `json:"summary"`
	Versions []catalogVersion `json:"versions"`
}
type catalogVersion struct {
	Version        string                      `json:"version"`
	ABI            string                      `json:"abi"`
	Implementation runnerv1.ImplementationKind `json:"implementation"`
	Behavior       runnerv1.ExecutionBehavior  `json:"behavior"`
	Capabilities   []runnerv1.Capability       `json:"capabilities"`
	UI             catalogUI                   `json:"ui"`
	Archive        catalogArchive              `json:"archive"`
}
type catalogUI struct {
	Name  string `json:"name"`
	Group string `json:"group"`
	Glyph string `json:"glyph,omitempty"`
}
type catalogArchive struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
type repositoryMetadata struct {
	Summary string `json:"summary"`
}

func main() {
	if len(os.Args) < 2 || os.Args[1] != "build" {
		fmt.Fprintln(os.Stderr, "usage: repository build -output DIRECTORY")
		os.Exit(2)
	}
	flags := flag.NewFlagSet("repository build", flag.ExitOnError)
	output := flags.String("output", "dist", "output directory")
	_ = flags.Parse(os.Args[2:])
	if err := build(context.Background(), *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func build(ctx context.Context, output string) error {
	if err := os.RemoveAll(output); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(output, "packages"), 0o755); err != nil {
		return err
	}
	templates, err := filepath.Glob("nodes/*/*/package.template.json")
	if err != nil {
		return err
	}
	if len(templates) == 0 {
		return errors.New("no node package templates found")
	}
	packages := map[string]*catalogPackage{}
	seen := map[string]struct{}{}
	for _, template := range templates {
		root := filepath.Dir(template)
		var raw struct {
			ID, Version    string
			Implementation runnerv1.ImplementationKind
			Entrypoint     string
		}
		data, err := os.ReadFile(template)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		key := raw.ID + "@" + raw.Version
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate package identity %s", key)
		}
		seen[key] = struct{}{}
		packageDir := filepath.Join(output, ".packages", strings.ReplaceAll(key, "/", "_"))
		if raw.Implementation == runnerv1.ImplementationWasm {
			if _, err := runnerpackage.BuildGo(ctx, runnerpackage.GoBuildOptions{ManifestTemplate: template, Source: "./" + filepath.ToSlash(filepath.Join(root, "source")), OutputDirectory: packageDir, WorkingDirectory: "."}); err != nil {
				return fmt.Errorf("build %s: %w", key, err)
			}
		} else {
			if _, err := runnerpackage.Assemble(ctx, template, filepath.Join(root, "source", raw.Entrypoint), packageDir); err != nil {
				return fmt.Errorf("assemble %s: %w", key, err)
			}
		}
		fileName := raw.ID + "-" + raw.Version + ".nodima-runner.zip"
		archivePath := filepath.Join(output, "packages", fileName)
		if err := runnerpackage.ArchiveDirectory(ctx, packageDir, archivePath); err != nil {
			return err
		}
		verified, err := runnerpackage.LoadArchive(archivePath)
		if err != nil {
			return err
		}
		archiveData, err := os.ReadFile(archivePath)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(archiveData)
		var metadata repositoryMetadata
		metadataData, err := os.ReadFile(filepath.Join(root, "repository.json"))
		if err != nil {
			return err
		}
		if err := json.Unmarshal(metadataData, &metadata); err != nil {
			return err
		}
		if strings.TrimSpace(metadata.Summary) == "" {
			return fmt.Errorf("%s requires a summary", key)
		}
		item := packages[raw.ID]
		if item == nil {
			item = &catalogPackage{ID: raw.ID, Summary: metadata.Summary, Versions: []catalogVersion{}}
			packages[raw.ID] = item
		}
		if item.Summary != metadata.Summary {
			return fmt.Errorf("%s versions must share one summary", raw.ID)
		}
		manifest := verified.Manifest
		item.Versions = append(item.Versions, catalogVersion{Version: manifest.Version, ABI: manifest.ABI, Implementation: manifest.Implementation, Behavior: manifest.Behavior, Capabilities: append([]runnerv1.Capability{}, manifest.Capabilities...), UI: catalogUI{Name: verified.UI.Name, Group: verified.UI.Group, Glyph: verified.UI.Glyph}, Archive: catalogArchive{URL: releaseBase + "/" + manifest.ID + "@" + manifest.Version + "/" + fileName, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(archiveData))}})
	}
	result := catalog{FormatVersion: "dbminer.node.repository.v1alpha1", Packages: make([]catalogPackage, 0, len(packages))}
	for _, item := range packages {
		sort.Slice(item.Versions, func(i, j int) bool { return item.Versions[i].Version < item.Versions[j].Version })
		result.Packages = append(result.Packages, *item)
	}
	sort.Slice(result.Packages, func(i, j int) bool { return result.Packages[i].ID < result.Packages[j].ID })
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	return os.WriteFile(filepath.Join(output, "catalog.json"), encoded, 0o644)
}
