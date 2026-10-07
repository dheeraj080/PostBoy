package interp

import (
	"errors"
	"regexp"
	"strconv"
	"testing"
	"time"
)

type fakeSecrets map[string]string

func (f fakeSecrets) Get(name string) (string, error) {
	if v, ok := f[name]; ok {
		return v, nil
	}
	return "", errors.New("not found")
}

func TestExpand(t *testing.T) {
	secrets := fakeSecrets{"TOKEN": "s3cr3t"}
	env := map[string]string{"BASE_URL": "https://api.test"}

	tests := []struct {
		in, want string
		mask     bool
	}{
		{"{{BASE_URL}}/users", "https://api.test/users", false},
		{"{{ .BASE_URL }}/x", "https://api.test/x", false},
		{"Bearer {{ secret.TOKEN }}", "Bearer s3cr3t", false},
		{"Bearer {{ secret.TOKEN }}", "Bearer ***", true},
		{"{{ secret.NOPE }}", MissingSecret, false},
		{"{{UNKNOWN}}", "{{UNKNOWN}}", false},
		{"no templates", "no templates", false},
	}
	for _, tt := range tests {
		if got := Expand(tt.in, env, secrets, tt.mask); got != tt.want {
			t.Errorf("Expand(%q, mask=%v) = %q, want %q", tt.in, tt.mask, got, tt.want)
		}
	}
}

func TestExpandNested(t *testing.T) {
	env := map[string]string{"HOST": "api.test", "BASE": "https://{{HOST}}/v1", "A": "{{B}}", "B": "{{A}}"}
	if got := Expand("{{BASE}}/x", env, nil, false); got != "https://api.test/v1/x" {
		t.Fatalf("nested = %q", got)
	}
	// Cycles terminate.
	_ = Expand("{{A}}", env, nil, false)
}

func TestExpandNestedSecretInVar(t *testing.T) {
	env := map[string]string{"AUTH": "Bearer {{ secret.TOKEN }}"}
	if got := Expand("{{AUTH}}", env, fakeSecrets{"TOKEN": "t"}, false); got != "Bearer t" {
		t.Fatalf("got %q", got)
	}
	if got := ExpandEnv("{{AUTH}}", env); got != "Bearer {{ secret.TOKEN }}" {
		t.Fatalf("ExpandEnv resolved a secret: %q", got)
	}
}

func TestDynamicVars(t *testing.T) {
	old := now
	now = func() time.Time { return time.Unix(1700000000, 0) }
	defer func() { now = old }()

	if got := Expand("{{$timestamp}}", nil, nil, false); got != "1700000000" {
		t.Fatalf("timestamp = %q", got)
	}
	if got := Expand("{{$isoTimestamp}}", nil, nil, false); got != "2023-11-14T22:13:20Z" {
		t.Fatalf("iso = %q", got)
	}
	u1, u2 := Expand("{{$uuid}}", nil, nil, false), Expand("{{$guid}}", nil, nil, false)
	uuidRe := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	if !uuidRe.MatchString(u1) || u1 == u2 {
		t.Fatalf("uuids = %q %q", u1, u2)
	}
	n, err := strconv.Atoi(Expand("{{$randomInt}}", nil, nil, false))
	if err != nil || n < 0 || n > 1000 {
		t.Fatalf("randomInt = %d %v", n, err)
	}
	if got := Expand("{{$nope}}", nil, nil, false); got != "{{$nope}}" {
		t.Fatalf("unknown dynamic = %q", got)
	}
}

func TestValidVarName(t *testing.T) {
	for name, want := range map[string]bool{"BASE_URL": true, "_x": true, "1A": false, "": false, "a-b": false} {
		if ValidVarName(name) != want {
			t.Errorf("ValidVarName(%q) != %v", name, want)
		}
	}
}

func TestExpandNilSecrets(t *testing.T) {
	if got := Expand("{{ secret.X }}", nil, nil, false); got != MissingSecret {
		t.Fatalf("got %q", got)
	}
}

func TestValidSecretName(t *testing.T) {
	for name, want := range map[string]bool{"TOKEN": true, "api_key_2": true, "": false, "a-b": false, "a b": false} {
		if ValidSecretName(name) != want {
			t.Errorf("ValidSecretName(%q) != %v", name, want)
		}
	}
}
