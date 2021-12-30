// Copyright 2019 Google Inc. All Rights Reserved.
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

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/golang/glog"
	"github.com/google/go-licenses/v2/gocli"
	"github.com/google/go-licenses/v2/licenses"
	"github.com/otiai10/copy"
	"github.com/spf13/cobra"
)

var (
	saveCmd = &cobra.Command{
		Use:   "save <package>",
		Short: "Saves licenses, copyright notices and source code, as required by a Go package's dependencies, to a directory.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  saveMain,
	}

	noticeRegexp = regexp.MustCompile(`^NOTICE(\.(txt|md))?$`)

	// savePath is where the output of the command is written to.
	savePath string
	// overwriteSavePath controls behaviour when the directory indicated by savePath already exists.
	// If true, the directory will be replaced. If false, the command will fail.
	overwriteSavePath bool
)

func init() {
	saveCmd.Flags().StringVar(&savePath, "save_path", "", "Directory into which files should be saved that are required by license terms")
	if err := saveCmd.MarkFlagRequired("save_path"); err != nil {
		glog.Fatal(err)
	}
	if err := saveCmd.MarkFlagFilename("save_path"); err != nil {
		glog.Fatal(err)
	}

	saveCmd.Flags().BoolVar(&overwriteSavePath, "force", false, "Delete the destination directory if it already exists.")

	rootCmd.AddCommand(saveCmd)
}

func saveMain(_ *cobra.Command, args []string) error {

	classifier, err := licenses.NewClassifier(confidenceThreshold)
	if err != nil {
		return err
	}

	mods, err := gocli.ListDeps(args...)
	if err != nil {
		return err
	}

	if overwriteSavePath {
		if err := os.RemoveAll(savePath); err != nil {
			return err
		}
	}

	// Check that the save path doesn't exist, otherwise it'd end up with a mix of
	// existing files and the output of this command.
	if d, err := os.Open(savePath); err == nil {
		d.Close()
		return fmt.Errorf("%s already exists", savePath)
	} else if !os.IsNotExist(err) {
		return err
	}

	modsWithBadLicenses := make(map[licenses.Type][]*licenses.Module)
	for _, m := range mods {
		mod, err := licenses.Scan(context.Background(), m, classifier, licenses.ScanOptions{})
		if err != nil {
			return err
		}
		modSaveDir := filepath.Join(savePath, mod.Path)
		// Detect what type of license this module has and fulfill its requirements, e.g. copy license, copyright notice, source code, etc.

		// Finds the most strict license type, defaults to unencumbered (the most permissive).
		// Note, len(mod.Licenses) > 0, because if mod does not have any
		// licenses, licenses.Scan will return an error and exit early.
		licenseType := licenses.Unencumbered
		for _, license := range mod.Licenses {
			if licenses.Stricter(license.Type, licenseType) {
				licenseType = license.Type
			}
		}

		// For simplicity, we pick the most strict license and comply
		// to all licenses in the same way.
		switch licenseType {
		case licenses.Restricted, licenses.Reciprocal:
			// Copy the entire source directory for the module.
			if err := copySrc(mod.Dir, modSaveDir); err != nil {
				return err
			}
		case licenses.Notice, licenses.Permissive, licenses.Unencumbered:
			// Just copy the license and copyright notice.
			if err := copyNotices(mod, modSaveDir); err != nil {
				return err
			}
		default:
			// Note, mod variable will keep changing, so clone it first.
			clonedMod := mod
			modsWithBadLicenses[licenseType] = append(modsWithBadLicenses[licenseType], &clonedMod)
		}
	}
	if len(modsWithBadLicenses) > 0 {
		return fmt.Errorf("one or more modules have an incompatible/unknown license: %q", modsWithBadLicenses)
	}
	return nil
}

// Dir permission needs execute bit for `cd` or `ls` commands
// ref: https://www.tutorialspoint.com/unix/unix-file-permission.htm
const permDirCurrentUser = 0700

var copyOpt = copy.Options{
	// Go module files are by default read-only, so we need to change perm on copy.
	// Reference: https://github.com/golang/go/issues/31481.
	AddPermission: 0600,
	// Skip the .git directory for copying, if it exists, since we don't want to save the user's
	// local Git config along with the source code.
	Skip: func(src string) (bool, error) { return strings.HasSuffix(src, ".git"), nil },
}

func copySrc(src, dest string) error {
	if err := copy.Copy(src, dest, copyOpt); err != nil {
		return err
	}
	return nil
}

func copyNotices(mod licenses.Module, dest string) error {
	saved := map[string]bool{}
	for _, license := range mod.Licenses {
		licensePath := filepath.Join(mod.Dir, license.Path)
		if err := copy.Copy(licensePath, filepath.Join(dest, license.Path), copyOpt); err != nil {
			return err
		}
		dir := filepath.Dir(licensePath)
		if saved[dir] {
			continue
		}
		saved[dir] = true
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			return err
		}
		dirRelativePath, err := filepath.Rel(mod.Dir, dir)
		if err != nil {
			return err
		}
		for _, f := range files {
			if fName := f.Name(); !f.IsDir() && noticeRegexp.MatchString(fName) {
				if err := copy.Copy(filepath.Join(dir, fName), filepath.Join(dest, dirRelativePath, fName), copyOpt); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
