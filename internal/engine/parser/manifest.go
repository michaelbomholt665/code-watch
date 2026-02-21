package parser

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

type Manifest struct {
	Version            int        `toml:"version"`
	AllowedAIBVersions []int      `toml:"allowed_aib_versions"`
	Artifacts          []Artifact `toml:"artifacts"`
}

type Artifact struct {
	Language        string `toml:"language"`
	AIBVersion      int    `toml:"aib_version"`
	SOPath          string `toml:"so_path"`
	SOSHA256        string `toml:"so_sha256"`
	NodeTypesPath   string `toml:"node_types_path"`
	NodeTypesSHA256 string `toml:"node_types_sha256"`
	Source          string `toml:"source"`
	ApprovedDate    string `toml:"approved_date"`
}

func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if _, err := toml.Decode(string(data), &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (m *Manifest) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(m)
}

func (m *Manifest) AddArtifact(art Artifact) {
	if art.ApprovedDate == "" {
		art.ApprovedDate = time.Now().Format("2006-01-02")
	}
	for i, existing := range m.Artifacts {
		if existing.Language == art.Language {
			m.Artifacts[i] = art
			return
		}
	}
	m.Artifacts = append(m.Artifacts, art)
}

func (m *Manifest) RemoveArtifact(language string) {
	out := make([]Artifact, 0, len(m.Artifacts))
	for _, art := range m.Artifacts {
		if art.Language != language {
			out = append(out, art)
		}
	}
	m.Artifacts = out
}

func CalculateSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
