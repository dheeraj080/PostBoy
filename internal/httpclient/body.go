package httpclient

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/dheeraj080/PostBoy/internal/config"
)

// MaxUploadBytes caps the total size of files attached to a request body.
const MaxUploadBytes = 100 << 20

// builtBody is a prepared request body.
type builtBody struct {
	reader        io.Reader
	length        int64  // -1 when unknown
	contentType   string // set by the body mode; "" leaves headers alone
	forceCT       bool   // contentType overrides any user Content-Type header
	defaultCTOnly bool   // contentType applies only if no header sets one
	closer        io.Closer
}

// ExpandPath expands a leading ~ to the user's home directory.
func ExpandPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[1:])
		}
	}
	return p
}

// buildBody prepares the body for req according to its BodyMode.
func (r Request) buildBody() (*builtBody, error) {
	if !SupportsBody(r.Method) {
		return nil, nil
	}
	switch r.BodyMode {
	case config.BodyRaw:
		if r.Body == "" {
			return nil, nil
		}
		s, err := r.expand(r.Body)
		if err != nil {
			return nil, fmt.Errorf("body: %w", err)
		}
		return &builtBody{reader: strings.NewReader(s), length: int64(len(s))}, nil

	case config.BodyURLEncoded:
		var pairs []string
		for _, f := range r.Form {
			k := strings.TrimSpace(f.Key)
			if !f.Enabled || k == "" {
				continue
			}
			v, err := r.expand(f.Value)
			if err != nil {
				return nil, fmt.Errorf("form field %s: %w", k, err)
			}
			pairs = append(pairs, url.QueryEscape(k)+"="+url.QueryEscape(v))
		}
		s := strings.Join(pairs, "&")
		return &builtBody{reader: strings.NewReader(s), length: int64(len(s)),
			contentType: "application/x-www-form-urlencoded", forceCT: true}, nil

	case config.BodyMultipart:
		return r.buildMultipart()

	case config.BodyFile:
		p, err := r.expand(r.BodyFile)
		if err != nil {
			return nil, fmt.Errorf("body file: %w", err)
		}
		p = ExpandPath(p)
		if p == "" {
			return nil, fmt.Errorf("body file: no file selected")
		}
		f, err := os.Open(p)
		if err != nil {
			return nil, fmt.Errorf("body file: %w", err)
		}
		st, err := f.Stat()
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("body file: %w", err)
		}
		if st.IsDir() {
			f.Close()
			return nil, fmt.Errorf("body file: %s is a directory", p)
		}
		ct := mime.TypeByExtension(filepath.Ext(p))
		if ct == "" {
			ct = "application/octet-stream"
		}
		return &builtBody{reader: f, length: st.Size(), contentType: ct, defaultCTOnly: true, closer: f}, nil
	}
	return nil, fmt.Errorf("unknown body mode %q", r.BodyMode)
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func (r Request) buildMultipart() (*builtBody, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	var total int64
	for _, f := range r.Form {
		k := strings.TrimSpace(f.Key)
		if !f.Enabled || k == "" {
			continue
		}
		v, err := r.expand(f.Value)
		if err != nil {
			return nil, fmt.Errorf("form field %s: %w", k, err)
		}
		if !f.File {
			if err := w.WriteField(k, v); err != nil {
				return nil, err
			}
			continue
		}
		path := ExpandPath(v)
		st, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("form file %s: %w", k, err)
		}
		if st.IsDir() {
			return nil, fmt.Errorf("form file %s: %s is a directory", k, path)
		}
		total += st.Size()
		if total > MaxUploadBytes {
			return nil, fmt.Errorf("form files exceed %d MB", MaxUploadBytes>>20)
		}
		ct := mime.TypeByExtension(filepath.Ext(path))
		if ct == "" {
			ct = "application/octet-stream"
		}
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			quoteEscaper.Replace(k), quoteEscaper.Replace(filepath.Base(path))))
		h.Set("Content-Type", ct)
		part, err := w.CreatePart(h)
		if err != nil {
			return nil, err
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("form file %s: %w", k, err)
		}
		_, err = io.Copy(part, file)
		file.Close()
		if err != nil {
			return nil, fmt.Errorf("form file %s: %w", k, err)
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return &builtBody{reader: bytes.NewReader(buf.Bytes()), length: int64(buf.Len()),
		contentType: w.FormDataContentType(), forceCT: true}, nil
}
