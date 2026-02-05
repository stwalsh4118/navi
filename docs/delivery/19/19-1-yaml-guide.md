# YAML v3 Guide for Task 19-1

**Date**: 2026-02-05
**Package**: `gopkg.in/yaml.v3`
**Documentation**: https://pkg.go.dev/gopkg.in/yaml.v3

## Installation

```bash
go get gopkg.in/yaml.v3
```

## Import

```go
import "gopkg.in/yaml.v3"
```

## Key Functions

### Unmarshal - Decode YAML to Go Values

```go
func Unmarshal(in []byte, out interface{}) error
```

Decodes the first YAML document and assigns values into the `out` parameter.

### Marshal - Encode Go Values to YAML

```go
func Marshal(in interface{}) ([]byte, error)
```

Serializes a Go value into a YAML document.

## Struct Tags

```go
type Config struct {
    Name       string `yaml:"name"`               // Custom key name
    Optional   string `yaml:"optional,omitempty"` // Omit if empty
    Ignored    int    `yaml:"-"`                  // Skip field
}
```

## Example for Our Use Case

```go
// RemoteConfig represents a single remote machine configuration
type RemoteConfig struct {
    Name        string `yaml:"name"`
    Host        string `yaml:"host"`
    User        string `yaml:"user"`
    Key         string `yaml:"key"`
    SessionsDir string `yaml:"sessions_dir,omitempty"`
    JumpHost    string `yaml:"jump_host,omitempty"`
}

// RemotesConfig is the root YAML structure
type RemotesConfig struct {
    Remotes []RemoteConfig `yaml:"remotes"`
}

// Loading from file
func loadConfig(path string) (*RemotesConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var config RemotesConfig
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, err
    }
    return &config, nil
}
```

## Error Handling

- Returns `nil` error on success
- Returns `*yaml.TypeError` for type mismatches (partial unmarshal may still occur)
- Returns `error` for malformed YAML
