// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package formatter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwt"

	jvspb "github.com/abcxyz/jvs/apis/v0"
)

// Text outputs the token into three sections:
//   - breakglass
//   - justification
//   - standard claims
type Text struct{}

// Ensure Text implements Formatter.
var _ Formatter = (*Text)(nil)

// NewText creates a new text formatter.
func NewText() *Text {
	return &Text{}
}

// FormatTo renders the token to the given writer as a table.
func (t *Text) FormatTo(ctx context.Context, w io.Writer, token jwt.Token, breakglass bool) error {
	if breakglass {
		if _, err := fmt.Fprintf(w, "\nWarning! This is a breakglass token.\n"); err != nil {
			return fmt.Errorf("failed to print breakglass warning: %w", err)
		}
	}

	// Write justifications
	if _, err := fmt.Fprintln(w); err != nil {
		return fmt.Errorf("failed to print newline: %w", err)
	}
	if err := t.writeHeader(w, "Justifications"); err != nil {
		return err
	}
	justifications, err := jvspb.GetJustifications(token)
	if err != nil {
		return fmt.Errorf("failed to get justifications from token: %w", err)
	}
	justificationClaims := make(map[string]string, len(justifications))
	for _, j := range justifications {
		justificationClaims[j.GetCategory()] = j.GetValue()
	}
	if err := t.writeTable(w, justificationClaims); err != nil {
		return fmt.Errorf("failed to write justifications: %w", err)
	}

	// Write other claims
	if _, err := fmt.Fprintln(w); err != nil {
		return fmt.Errorf("failed to print newline: %w", err)
	}
	if err := t.writeHeader(w, "Claims"); err != nil {
		return err
	}
	standard, err := token.AsMap(ctx)
	if err != nil {
		return fmt.Errorf("failed to convert token claims into map: %w", err)
	}
	delete(standard, jvspb.JustificationsKey)
	standardClaims := make(map[string]string, len(standard))
	for k, v := range standard {
		str, err := t.bestStringRepresentation(v)
		if err != nil {
			return fmt.Errorf("failed to convert to string: %w", err)
		}
		standardClaims[k] = str
	}
	if err := t.writeTable(w, standardClaims); err != nil {
		return fmt.Errorf("failed to write standard claims: %w", err)
	}

	return nil
}

// writeHeader writes a single header entry with a trailing newline character.
func (t *Text) writeHeader(w io.Writer, header string) error {
	if _, err := fmt.Fprintf(w, "----- %s -----\n", header); err != nil {
		return fmt.Errorf("failed to write header for %q: %w", header, err)
	}
	return nil
}

// writeTable generates a table for the supplied key-value entries. The keys are
// sorted lexographically before printing.
func (t *Text) writeTable(w io.Writer, entries map[string]string) error {
	keys := make([]string, 0, len(entries))
	longest := 0
	for key := range entries {
		if l := len(key); l > longest {
			longest = l
		}

		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := entries[key]
		if _, err := fmt.Fprintf(w, "%-*s%s\n", longest+4, key, value); err != nil {
			return fmt.Errorf("failed to write key %q: %w", key, err)
		}
	}

	return nil
}

// bestStringRepresentation compiles the best string version of the given input.
func (t *Text) bestStringRepresentation(i any) (string, error) {
	v := reflect.ValueOf(i)

	// Resolve pointers.
	for v.Type().Kind() == reflect.Ptr {
		if v.IsNil() {
			return "", nil
		}
		v = v.Elem()
	}

	typ := v.Type()
	knd := typ.Kind()

	//nolint:exhaustive // We only want to support the given types
	switch knd {
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', 2, 64), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.Int64:
		// Special case time.Duration
		if typ.PkgPath() == "time" && typ.Name() == "Duration" {
			return time.Duration(v.Int()).Truncate(time.Second).String(), nil
		}
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.String:
		return v.String(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10), nil
	case reflect.Map:
		list := make([]string, 0, v.Len())
		for _, k := range v.MapKeys() {
			key, err := t.bestStringRepresentation(k.Interface())
			if err != nil {
				return "", err
			}
			val, err := t.bestStringRepresentation(v.MapIndex(k).Interface())
			if err != nil {
				return "", err
			}
			list = append(list, key+":"+val)
		}
		return "[" + strings.Join(list, ", ") + "]", nil
	case reflect.Array, reflect.Slice:
		// Special case []byte
		if typ.Elem().Kind() == reflect.Uint8 {
			return string(v.Bytes()), nil
		}

		list := make([]string, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			val, err := t.bestStringRepresentation(v.Index(i).Interface())
			if err != nil {
				return "", err
			}
			list = append(list, val)
		}
		return "[" + strings.Join(list, ", ") + "]", nil
	case reflect.Struct:
		// Special case time.Time
		if typ.PkgPath() == "time" && typ.Name() == "Time" {
			astime, ok := v.Interface().(time.Time)
			if !ok {
				return "", fmt.Errorf("could not convert to time.Time (this is a bug)")
			}
			astime = astime.UTC()
			return astime.Format("2006-01-02 3:04PM MST"), nil
		}

		b, err := json.Marshal(v.Interface())
		if err != nil {
			return "", fmt.Errorf("failed to make structure json: %w", err)
		}
		return string(b), nil
	default:
		return "", fmt.Errorf("unsupported interface %q for %#v", knd, i)
	}
}
