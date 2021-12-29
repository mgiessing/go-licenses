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
	"os"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"github.com/google/go-licenses/v2/licenses"
	"github.com/google/go-licenses/v2/third_party/go/pkgsite/source"
	"github.com/spf13/cobra"
)

var (
	csvCmd = &cobra.Command{
		Use:   "csv <package>",
		Short: "Prints all licenses that apply to a Go package and its dependencies",
		Args:  cobra.MinimumNArgs(1),
		RunE:  csvMain,
	}

	gitRemotes []string
)

func init() {
	csvCmd.Flags().StringArrayVar(&gitRemotes, "git_remote", []string{"origin", "upstream"}, "Remote Git repositories to try")

	rootCmd.AddCommand(csvCmd)
}

func csvMain(_ *cobra.Command, args []string) error {
	classifier, err := licenses.NewClassifier(confidenceThreshold)
	if err != nil {
		return err
	}

	libs, err := licenses.Libraries(context.Background(), classifier, args...)
	if err != nil {
		return err
	}
	for _, lib := range libs {
		m := lib.Module
		client := source.NewClient(time.Second * 20)
		ver := m.Version
		if ver == "" {
			// This always happens for the module in development.
			ver = "master"
			glog.Warningf("module %s has empty version, defaults to master. The license URL may be incorrect. Please verify!", m.Path)
		}
		remote, err := source.ModuleInfo(context.Background(), client, m.Path, ver)
		if err != nil {
			glog.Warningf("finding module info for %s: %s", m.Path, err)
			// Do not exit early, because URL is optional
			remote = nil
		}
		licenseName := "Unknown"
		licenseURL := "Unknown"
		if lib.LicensePath != "" {
			licenseName, _, err = classifier.Identify(lib.LicensePath)
			if err != nil {
				glog.Errorf("Error identifying license in %q: %v", lib.LicensePath, err)
				licenseName = "Unknown"
			}
			licenseRelativePath, err := filepath.Rel(m.Dir, lib.LicensePath)
			if err != nil {
				glog.Errorf("Error converting license path %q to module %s relative path: %v", lib.LicensePath, m.Path, err)
			}
			licenseURL = remote.FileURL(licenseRelativePath)
			if licenseURL == "" {
				licenseURL = "Unknown"
			}
		}
		if _, err := os.Stdout.WriteString(fmt.Sprintf("%s, %s, %s\n", lib.Name(), licenseURL, licenseName)); err != nil {
			return err
		}
	}
	return nil
}
