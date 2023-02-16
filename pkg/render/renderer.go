// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package render

import (
	"bytes"
	"context"
	"fmt"
	htmltemplate "html/template"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/abcxyz/pkg/logging"
	"go.uber.org/zap"
)

// allowedResponseCodes are the list of allowed response codes. This is
// primarily here to catch if someone, in the future, accidentally includes a
// bad status code.
var allowedResponseCodes = map[int]struct{}{
	http.StatusOK:         {},
	http.StatusBadRequest: {},
	// TODO add more response codes and render generic html for each
}

// Renderer is responsible for rendering various content and templates like HTML
// and JSON responses. This implementation caches templates and uses a pool of buffers.
type Renderer struct {
	// debug indicates templates should be reloaded on each invocation and real
	// error responses should be rendered. Do not enable in production.
	debug bool

	// logger is the log writer.
	logger *zap.SugaredLogger

	// rendererPool is a pool of *bytes.Buffer, used as a rendering buffer to
	// prevent partial responses being sent to clients.
	rendererPool *sync.Pool

	// templates is the actually collection of templates. templatesLock is a mutex to prevent
	// concurrent modification of the templates field.
	templates     *htmltemplate.Template
	templatesLock sync.RWMutex

	fs fs.FS
}

// New creates a new renderer with the given details.
func NewRenderer(ctx context.Context, fsys fs.FS, debug bool) (*Renderer, error) {
	logger := logging.FromContext(ctx)

	r := &Renderer{
		debug:  debug,
		logger: logger,
		rendererPool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, 1024))
			},
		},
		fs: fsys,
	}

	// Load initial templates
	if err := r.loadTemplates(); err != nil {
		return nil, err
	}

	return r, nil
}

// executeHTMLTemplate executes a single HTML template with the provided data.
func (r *Renderer) executeHTMLTemplate(w io.Writer, name string, data interface{}) error {
	r.templatesLock.RLock()
	defer r.templatesLock.RUnlock()

	if r.templates == nil {
		return fmt.Errorf("no html templates are defined")
	}

	if err := r.templates.ExecuteTemplate(w, name, data); err != nil {
		return fmt.Errorf("error with executeTemplate: %w", err)
	}

	return nil
}

// loadTemplates loads or reloads all templates.
func (r *Renderer) loadTemplates() error {
	r.templatesLock.Lock()
	defer r.templatesLock.Unlock()

	if r.fs == nil {
		return nil
	}

	htmltmpl := htmltemplate.New("").
		Option("missingkey=zero").
		Funcs(r.templateFuncs())

	if err := loadTemplates(r.fs, htmltmpl); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	r.templates = htmltmpl

	return nil
}

func loadTemplates(fsys fs.FS, htmltmpl *htmltemplate.Template) error {
	err := fs.WalkDir(fsys, ".", func(pth string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(info.Name(), ".html.tmpl") {
			if _, err := htmltmpl.ParseFS(fsys, pth); err != nil {
				return fmt.Errorf("failed to parse %s: %w", pth, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error while walking the directory: %w", err)
	}

	return nil
}

// Define helper methods that may be needed within the templates, examples can be found here
// https://github.com/google/exposure-notifications-verification-server/blob/main/pkg/render/renderer.go#L348-L385.
func (r *Renderer) templateFuncs() htmltemplate.FuncMap {
	return map[string]interface{}{
		// only pulling in js for popup page
		"jsPopupIncludeTag": assetIncludeTag(r.fs, "static/js/popup", jsPopupIncludeTmpl, &jsPopupIncludeTagCache, r.debug),
		// pull all css
		"cssIncludeTag": assetIncludeTag(r.fs, "static/css", cssIncludeTmpl, &cssIncludeTagCache, r.debug),

		"pathEscape":    url.PathEscape,
		"pathUnescape":  url.PathUnescape,
		"queryEscape":   url.QueryEscape,
		"queryUnescape": url.QueryUnescape,
	}
}

// AllowedResponseCode returns true if the code is a permitted response code,
// false otherwise.
func (r *Renderer) AllowedResponseCode(code int) bool {
	_, ok := allowedResponseCodes[code]
	return ok
}
