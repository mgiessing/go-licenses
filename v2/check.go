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

	"github.com/google/go-licenses/v2/licenses"
	"github.com/spf13/cobra"
)

var (
	checkCmd = &cobra.Command{
		Use:   "check <package>",
		Short: "Checks whether licenses for a package are not Forbidden.",
		Args:  cobra.MinimumNArgs(1),
		RunE:  checkMain,
	}
)

func init() {
	rootCmd.AddCommand(checkCmd)
}

func checkMain(_ *cobra.Command, args []string) error {
	classifier, err := licenses.NewClassifier(confidenceThreshold)
	if err != nil {
		return err
	}

	mods, err := licenses.Modules(context.Background(), classifier, args...)
	if err != nil {
		return err
	}
	for _, mod := range mods {
		for _, license := range mod.Licenses {
			if license.Type == licenses.Forbidden {
				fmt.Fprintf(os.Stderr, "Forbidden license type %s for library %v: %s\n", license.ID, mod.Path, license.URL)
				os.Exit(1)
			}
		}
	}
	return nil
}
