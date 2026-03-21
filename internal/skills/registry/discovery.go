package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type DiscoverySource interface {
	Name() string
	Discover(ctx context.Context) ([]*SkillManifest, error)
	Watch(ctx context.Context, callback func(*SkillManifest)) (Unsubscribe, error)
}

type LocalDiscovery struct {
	Paths []string
}

func NewLocalDiscovery(paths []string) *LocalDiscovery {
	return &LocalDiscovery{Paths: paths}
}

func (d *LocalDiscovery) Name() string {
	return "local"
}

func (d *LocalDiscovery) Discover(ctx context.Context) ([]*SkillManifest, error) {
	var manifests []*SkillManifest

	for _, path := range d.Paths {
		err := filepath.WalkDir(path, func(filePath string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if entry.IsDir() {
				return nil
			}

			if filepath.Base(filePath) == "skill.yaml" {
				manifest, err := loadManifest(filePath)
				if err != nil {
					return nil
				}
				manifest.Source = "local"
				manifest.FilePath = filePath
				manifests = append(manifests, manifest)
			}

			return nil
		})

		if err != nil && ctx.Err() == nil {
			return nil, err
		}
	}

	return manifests, ctx.Err()
}

func (d *LocalDiscovery) Watch(ctx context.Context, callback func(*SkillManifest)) (Unsubscribe, error) {
	cancel := make(chan struct{})

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-cancel:
				return
			case <-ticker.C:
				manifests, err := d.Discover(ctx)
				if err != nil {
					continue
				}
				for _, manifest := range manifests {
					callback(manifest)
				}
			}
		}
	}()

	return func() { close(cancel) }, nil
}

type RegistryDiscovery struct {
	URL    string
	Client *http.Client
}

func NewRegistryDiscovery(url string) *RegistryDiscovery {
	return &RegistryDiscovery{
		URL: url,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (d *RegistryDiscovery) Name() string {
	return "registry"
}

func (d *RegistryDiscovery) Discover(ctx context.Context) ([]*SkillManifest, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", d.URL+"/v1/skills", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := d.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch skills: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var result struct {
		Skills []*SkillManifest `json:"skills"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	for _, manifest := range result.Skills {
		manifest.Source = d.URL
	}

	return result.Skills, nil
}

func (d *RegistryDiscovery) Watch(ctx context.Context, callback func(*SkillManifest)) (Unsubscribe, error) {
	cancel := make(chan struct{})

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-cancel:
				return
			case <-ticker.C:
				manifests, err := d.Discover(ctx)
				if err != nil {
					continue
				}
				for _, manifest := range manifests {
					callback(manifest)
				}
			}
		}
	}()

	return func() { close(cancel) }, nil
}

type GitAuth struct {
	Token  string
	SSHKey string
}

type GitRepoSource struct {
	URL    string
	Branch string
	Path   string
	Auth   *GitAuth
}

type GitDiscovery struct {
	Repos    []GitRepoSource
	CacheDir string
	fetchMu  sync.Mutex
}

func NewGitDiscovery(repos []GitRepoSource, cacheDir string) *GitDiscovery {
	return &GitDiscovery{
		Repos:    repos,
		CacheDir: cacheDir,
	}
}

func (d *GitDiscovery) Name() string {
	return "git"
}

func (d *GitDiscovery) Discover(ctx context.Context) ([]*SkillManifest, error) {
	var manifests []*SkillManifest

	for _, repo := range d.Repos {
		repoManifests, err := d.discoverRepo(ctx, repo)
		if err != nil && ctx.Err() == nil {
			continue
		}
		manifests = append(manifests, repoManifests...)
	}

	return manifests, ctx.Err()
}

func (d *GitDiscovery) discoverRepo(ctx context.Context, repo GitRepoSource) ([]*SkillManifest, error) {
	d.fetchMu.Lock()
	defer d.fetchMu.Unlock()

	cacheDir := filepath.Join(d.CacheDir, strings.ReplaceAll(repo.URL, "/", "_"))
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}

	skillDir := filepath.Join(cacheDir, repo.Path)

	var files []string
	err := filepath.WalkDir(skillDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && filepath.Base(path) == "skill.yaml" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var manifests []*SkillManifest
	for _, f := range files {
		manifest, err := loadManifest(f)
		if err != nil {
			continue
		}
		manifest.Source = repo.URL
		manifest.FilePath = f
		manifests = append(manifests, manifest)
	}

	return manifests, nil
}

func (d *GitDiscovery) Watch(ctx context.Context, callback func(*SkillManifest)) (Unsubscribe, error) {
	cancel := make(chan struct{})

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-cancel:
				return
			case <-ticker.C:
				manifests, err := d.Discover(ctx)
				if err != nil {
					continue
				}
				for _, manifest := range manifests {
					callback(manifest)
				}
			}
		}
	}()

	return func() { close(cancel) }, nil
}

type CompositeDiscovery struct {
	sources []DiscoverySource
}

func NewCompositeDiscovery(sources []DiscoverySource) *CompositeDiscovery {
	return &CompositeDiscovery{sources: sources}
}

func (d *CompositeDiscovery) Name() string {
	return "composite"
}

func (d *CompositeDiscovery) Discover(ctx context.Context) ([]*SkillManifest, error) {
	var allManifests []*SkillManifest
	var mu sync.Mutex

	var wg sync.WaitGroup
	errCh := make(chan error, len(d.sources))

	for _, source := range d.sources {
		wg.Add(1)
		go func(s DiscoverySource) {
			defer wg.Done()

			manifests, err := s.Discover(ctx)
			if err != nil {
				errCh <- err
				return
			}

			mu.Lock()
			allManifests = append(allManifests, manifests...)
			mu.Unlock()
		}(source)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil && ctx.Err() == nil {
			return nil, err
		}
	}

	return allManifests, ctx.Err()
}

func (d *CompositeDiscovery) Watch(ctx context.Context, callback func(*SkillManifest)) (Unsubscribe, error) {
	unsubs := make([]Unsubscribe, len(d.sources))

	for i, source := range d.sources {
		unsub, err := source.Watch(ctx, callback)
		if err != nil {
			continue
		}
		unsubs[i] = unsub
	}

	return func() {
		for _, unsub := range unsubs {
			if unsub != nil {
				unsub()
			}
		}
	}, nil
}

func loadManifest(filePath string) (*SkillManifest, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest SkillManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	if manifest.APIVersion != "swarm.ai/v1" || manifest.Kind != "Skill" {
		return nil, fmt.Errorf("invalid manifest: expected apiVersion=swarm.ai/v1, kind=Skill")
	}

	if manifest.Metadata.Name == "" {
		return nil, fmt.Errorf("manifest missing required field: metadata.name")
	}

	if manifest.Metadata.Version == "" {
		return nil, fmt.Errorf("manifest missing required field: metadata.version")
	}

	if manifest.Metadata.Description == "" {
		return nil, fmt.Errorf("manifest missing required field: metadata.description")
	}

	if manifest.Spec.Runtime == "" {
		return nil, fmt.Errorf("manifest missing required field: spec.runtime")
	}

	if manifest.Spec.Entrypoint == "" {
		return nil, fmt.Errorf("manifest missing required field: spec.entrypoint")
	}

	return &manifest, nil
}

func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := max(len(parts1), len(parts2))
	for i := 0; i < maxLen; i++ {
		var num1, num2 int
		if i < len(parts1) {
			fmt.Sscanf(parts1[i], "%d", &num1)
		}
		if i < len(parts2) {
			fmt.Sscanf(parts2[i], "%d", &num2)
		}

		if num1 < num2 {
			return -1
		}
		if num1 > num2 {
			return 1
		}
	}

	return 0
}
