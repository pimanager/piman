package deploy

import (
	"bytes"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParsePortOverride(t *testing.T) {
	tests := []struct {
		input    string
		expected PortOverride
		wantErr  bool
	}{
		{
			input: "8080:80",
			expected: PortOverride{
				HostPort:      "8080",
				ContainerPort: "80",
			},
			wantErr: false,
		},
		{
			input: "web:8080:80",
			expected: PortOverride{
				Service:       "web",
				HostPort:      "8080",
				ContainerPort: "80",
			},
			wantErr: false,
		},
		{
			input:   "invalid",
			wantErr: true,
		},
		{
			input:   ":80",
			wantErr: true,
		},
		{
			input:   "8080:",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		got, err := ParsePortOverride(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParsePortOverride(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr {
			if got.Service != tt.expected.Service || got.HostPort != tt.expected.HostPort || got.ContainerPort != tt.expected.ContainerPort {
				t.Errorf("ParsePortOverride(%q) = %+v, want %+v", tt.input, got, tt.expected)
			}
		}
	}
}

func TestCustomizeComposePorts(t *testing.T) {
	tests := []struct {
		name      string
		yamlInput string
		overrides []PortOverride
		wantYAML  string
		wantErr   bool
	}{
		{
			name: "Simple string port update",
			yamlInput: `
services:
  web:
    image: nginx
    ports:
      - "80:80"
`,
			overrides: []PortOverride{
				{HostPort: "8080", ContainerPort: "80"},
			},
			wantYAML: `
services:
  web:
    image: nginx
    ports:
      - "8080:80"
`,
		},
		{
			name: "Integer port update",
			yamlInput: `
services:
  web:
    image: nginx
    ports:
      - 80
`,
			overrides: []PortOverride{
				{HostPort: "8080", ContainerPort: "80"},
			},
			wantYAML: `
services:
  web:
    image: nginx
    ports:
      - "8080:80"
`,
		},
		{
			name: "Long syntax port update",
			yamlInput: `
services:
  web:
    image: nginx
    ports:
      - target: 80
        published: 80
        protocol: tcp
`,
			overrides: []PortOverride{
				{HostPort: "8080", ContainerPort: "80"},
			},
			wantYAML: `
services:
  web:
    image: nginx
    ports:
      - target: 80
        published: 8080
        protocol: tcp
`,
		},
		{
			name: "Add port mapping when not present",
			yamlInput: `
services:
  web:
    image: nginx
`,
			overrides: []PortOverride{
				{HostPort: "8080", ContainerPort: "80"},
			},
			wantYAML: `
services:
  web:
    image: nginx
    ports:
      - "8080:80"
`,
		},
		{
			name: "Target specific service",
			yamlInput: `
services:
  web:
    image: nginx
  db:
    image: mysql
`,
			overrides: []PortOverride{
				{Service: "web", HostPort: "8080", ContainerPort: "80"},
			},
			wantYAML: `
services:
  web:
    image: nginx
    ports:
      - "8080:80"
  db:
    image: mysql
`,
		},
		{
			name: "Error on ambiguous target service",
			yamlInput: `
services:
  web:
    image: nginx
  db:
    image: mysql
`,
			overrides: []PortOverride{
				{HostPort: "8080", ContainerPort: "80"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CustomizeComposePorts([]byte(tt.yamlInput), tt.overrides)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CustomizeComposePorts() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			// Parse both and compare as maps to avoid formatting/whitespace differences
			var gotMap, wantMap map[string]interface{}
			if err := yaml.Unmarshal(got, &gotMap); err != nil {
				t.Fatalf("failed to parse got yaml: %v", err)
			}
			if err := yaml.Unmarshal([]byte(tt.wantYAML), &wantMap); err != nil {
				t.Fatalf("failed to parse want yaml: %v", err)
			}

			gotBytes, _ := yaml.Marshal(gotMap)
			wantBytes, _ := yaml.Marshal(wantMap)
			if !bytes.Equal(gotBytes, wantBytes) {
				t.Errorf("CustomizeComposePorts() = \n%s\nwant:\n%s", string(gotBytes), string(wantBytes))
			}
		})
	}
}
