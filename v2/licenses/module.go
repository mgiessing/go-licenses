// Copyright 2021 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package licenses

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/go-licenses/v2/gocli"
)

type License struct {
	ID   string // SPDX ID. https://spdx.org/licenses.
	Path string // Relative path in the module.
	URL  string // Optional, license file URL on internet.
}

type Module struct {
	gocli.Module
	Licenses []License
}

// Modules finds licenses of direct and transitive module dependencies of the import path packages.
func Modules(classifier Classifier, importPaths ...string) ([]Module, error) {
	mods, err := gocli.ListDeps(importPaths...)
	if err != nil {
		return nil, err
	}
	res := make([]Module, 0, len(mods))
	for _, mod := range mods {
		modLicense, err := module(mod, classifier)
		if err != nil {
			return res, err
		}
		res = append(res, modLicense)
	}
	return res, nil
}

var ErrorEmptyDir = fmt.Errorf("dir is empty")

// module scans a module for licenses.
func module(m gocli.Module, classifier Classifier) (res Module, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("scanning licenses for module %q: %w", m.Path, err)
		}
	}()
	res.Module = m
	if m.Dir == "" {
		return res, ErrorEmptyDir
	}
	res.Licenses = make([]License, 0)
	err = filepath.Walk(m.Dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			// skip symbolic links
			return nil
		}
		if info.IsDir() {
			if ignoredDir[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !licenseRegexp.MatchString(info.Name()) {
			// Skip file names that does not look like a license file.
			return nil
		}
		licenseID, _, err := classifier.Identify(path)
		if err != nil {
			// It's expected for files without license text in it.
			return nil
		}
		res.Licenses = append(res.Licenses, License{
			ID:   licenseID,
			Path: path,
		})
		return nil
	})
	if len(res.Licenses) == 0 {
		return res, fmt.Errorf("license not found")
	}
	return res, err
}

var ignoredDir map[string]bool = make(map[string]bool)

func init() {
	ignoredDir[".git"] = true
	ignoredDir["node_modules"] = true
	ignoredDir["testdata"] = true
}
