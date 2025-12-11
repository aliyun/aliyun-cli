package plugin

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

const (
	IndexURL = "https://aliyun-cli-pub.oss-cn-hangzhou.aliyuncs.com/plugins/plugin-index.json" // 默认索引地址
)

type Manager struct {
	rootDir string
}

func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	rootDir := filepath.Join(home, ".aliyun", "plugins")
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, err
	}
	return &Manager{rootDir: rootDir}, nil
}

func (m *Manager) GetIndex() (*Index, error) {
	resp, err := http.Get(IndexURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch plugin index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch plugin index: status %d", resp.StatusCode)
	}

	var index Index
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, fmt.Errorf("failed to decode plugin index: %w", err)
	}
	return &index, nil
}

func (m *Manager) GetLocalManifest() (*LocalManifest, error) {
	path := filepath.Join(m.rootDir, "manifest.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &LocalManifest{Plugins: make(map[string]LocalPlugin)}, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var manifest LocalManifest
	if err := json.NewDecoder(f).Decode(&manifest); err != nil {
		return nil, err
	}
	if manifest.Plugins == nil {
		manifest.Plugins = make(map[string]LocalPlugin)
	}
	return &manifest, nil
}

func (m *Manager) saveLocalManifest(manifest *LocalManifest) error {
	path := filepath.Join(m.rootDir, "manifest.json")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(manifest)
}

func (m *Manager) Install(pluginName, version string) error {
	index, err := m.GetIndex()
	if err != nil {
		return err
	}

	var targetPlugin *PluginInfo
	for _, p := range index.Plugins {
		if p.Name == pluginName {
			targetPlugin = &p
			break
		}
	}
	if targetPlugin == nil {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	if version == "" {
		version = targetPlugin.LatestVersion
	}

	verInfoMap, ok := targetPlugin.Versions[version]
	if !ok {
		return fmt.Errorf("version %s not found for plugin %s", version, pluginName)
	}

	platform := GetCurrentPlatform()
	platInfo, ok := verInfoMap[platform]
	if !ok {
		return fmt.Errorf("plugin %s version %s not supported on %s", pluginName, version, platform)
	}

	downloadURL := platInfo.URL

	// 下载并解压
	cli.Printf(cli.DefaultStdoutWriter(), "Downloading %s %s...\n", pluginName, version)

	tmpDir, err := os.MkdirTemp("", "aliyun-plugin-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, "plugin.archive")
	if err := downloadFile(downloadURL, archivePath); err != nil {
		return err
	}

	extractDir := filepath.Join(m.rootDir, pluginName)
	if err := os.RemoveAll(extractDir); err != nil {
		return err
	}
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return err
	}

	if strings.HasSuffix(downloadURL, ".zip") {
		if err := unzip(archivePath, extractDir); err != nil {
			return err
		}
	} else {
		if err := untar(archivePath, extractDir); err != nil {
			return err
		}
	}

	pluginManifestPath := filepath.Join(extractDir, "manifest.json")
	pluginManifestFile, err := os.Open(pluginManifestPath)
	if err != nil {
		return fmt.Errorf("invalid plugin package: manifest.json not found")
	}
	defer pluginManifestFile.Close()

	var pManifest PluginManifest
	if err := json.NewDecoder(pluginManifestFile).Decode(&pManifest); err != nil {
		return fmt.Errorf("invalid plugin manifest: %w", err)
	}

	localManifest, err := m.GetLocalManifest()
	if err != nil {
		return err
	}

	localManifest.Plugins[pluginName] = LocalPlugin{
		Name:        pluginName,
		Version:     version,
		Path:        extractDir,
		Command:     pManifest.Command,
		Description: pManifest.ShortDescription,
	}

	if err := m.saveLocalManifest(localManifest); err != nil {
		return err
	}

	cli.Printf(cli.DefaultStdoutWriter(), "Plugin %s %s installed successfully!\n", pluginName, version)
	return nil
}

func (m *Manager) Upgrade(pluginName string) error {
	localManifest, err := m.GetLocalManifest()
	if err != nil {
		return err
	}

	localPlugin, ok := localManifest.Plugins[pluginName]
	if !ok {
		return fmt.Errorf("plugin %s not installed", pluginName)
	}

	index, err := m.GetIndex()
	if err != nil {
		return err
	}

	var targetPlugin *PluginInfo
	for _, p := range index.Plugins {
		if p.Name == pluginName {
			targetPlugin = &p
			break
		}
	}
	if targetPlugin == nil {
		return fmt.Errorf("plugin %s not found in repository", pluginName)
	}

	if localPlugin.Version == targetPlugin.LatestVersion {
		cli.Printf(cli.DefaultStdoutWriter(), "Plugin %s is already up to date (version %s).\n", pluginName, localPlugin.Version)
		return nil
	}

	cli.Printf(cli.DefaultStdoutWriter(), "Upgrading plugin %s from %s to %s...\n", pluginName, localPlugin.Version, targetPlugin.LatestVersion)
	return m.Install(pluginName, targetPlugin.LatestVersion)
}

func (m *Manager) Uninstall(pluginName string) error {
	localManifest, err := m.GetLocalManifest()
	if err != nil {
		return err
	}

	plugin, ok := localManifest.Plugins[pluginName]
	if !ok {
		return fmt.Errorf("plugin %s not installed", pluginName)
	}

	if err := os.RemoveAll(plugin.Path); err != nil {
		return fmt.Errorf("failed to remove plugin files: %w", err)
	}

	delete(localManifest.Plugins, pluginName)
	if err := m.saveLocalManifest(localManifest); err != nil {
		return err
	}

	cli.Printf(cli.DefaultStdoutWriter(), "Plugin %s uninstalled.\n", pluginName)
	return nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func untar(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
