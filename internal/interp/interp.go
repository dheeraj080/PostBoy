// Package interp expands {{VAR}}, {{$dynamic}} and {{ secret.NAME }}
// templates.
package interp

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MissingSecret is substituted for secrets that cannot be resolved.
const MissingSecret = "[MISSING_SECRET]"

// maxDepth bounds nested variable expansion (and breaks reference cycles).
const maxDepth = 5

// SecretGetter resolves secret values by name.
type SecretGetter interface {
	Get(name string) (string, error)
}

var (
	templateRegex   = regexp.MustCompile(`\{\{\s*(?:secret[\.\s"]+([a-zA-Z0-9_]+)"?|\.?(\$?[a-zA-Z0-9_]+))\s*\}\}`)
	secretNameRegex = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	varNameRegex    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// ValidSecretName reports whether name may be used as a secret name.
func ValidSecretName(name string) bool { return secretNameRegex.MatchString(name) }

// ValidVarName reports whether name may be used as an environment variable.
func ValidVarName(name string) bool { return varNameRegex.MatchString(name) }

// DynamicVars lists the supported {{$name}} variables and their meaning.
var DynamicVars = []struct{ Name, Desc string }{
	{"$uuid", "random UUID v4 (alias: $guid, $randomUUID)"},
	{"$timestamp", "current Unix time in seconds"},
	{"$timestampMs", "current Unix time in milliseconds"},
	{"$isoTimestamp", "current UTC time, RFC 3339"},
	{"$randomInt", "random integer 0-1000"},
	{"$randomHex", "16 random hex characters"},
}

// now is overridable in tests.
var now = time.Now

func dynamic(name string) (string, bool) {
	switch name {
	case "$uuid", "$guid", "$randomUUID":
		return newUUID(), true
	case "$timestamp":
		return strconv.FormatInt(now().Unix(), 10), true
	case "$timestampMs":
		return strconv.FormatInt(now().UnixMilli(), 10), true
	case "$isoTimestamp":
		return now().UTC().Format(time.RFC3339), true
	case "$randomInt":
		n, err := rand.Int(rand.Reader, big.NewInt(1001))
		if err != nil {
			return "0", true
		}
		return n.String(), true
	case "$randomHex":
		var b [8]byte
		_, _ = rand.Read(b[:])
		return hex.EncodeToString(b[:]), true
	}
	return "", false
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Expand replaces template references in input. Variables may reference
// other variables (up to a fixed depth). Unknown variables are left
// untouched; unresolvable secrets become MissingSecret. When mask is true,
// resolved secrets render as "***".
func Expand(input string, env map[string]string, secrets SecretGetter, mask bool) string {
	return expand(input, env, secrets, mask, true, 0)
}

// ExpandEnv expands environment and dynamic variables but leaves secret
// references untouched. Use it when the result may be shared (e.g. "copy as
// curl").
func ExpandEnv(input string, env map[string]string) string {
	return expand(input, env, nil, false, false, 0)
}

func expand(input string, env map[string]string, secrets SecretGetter, mask, resolveSecrets bool, depth int) string {
	if depth > maxDepth || !strings.Contains(input, "{{") {
		return input
	}
	return templateRegex.ReplaceAllStringFunc(input, func(match string) string {
		sub := templateRegex.FindStringSubmatch(match)
		if sub[1] != "" {
			if !resolveSecrets {
				return match
			}
			if secrets == nil {
				return MissingSecret
			}
			val, err := secrets.Get(sub[1])
			if err != nil {
				return MissingSecret
			}
			if mask {
				return "***"
			}
			return val
		}
		name := sub[2]
		if strings.HasPrefix(name, "$") {
			if v, ok := dynamic(name); ok {
				return v
			}
			return match
		}
		if val, ok := env[name]; ok {
			return expand(val, env, secrets, mask, resolveSecrets, depth+1)
		}
		return match
	})
}
