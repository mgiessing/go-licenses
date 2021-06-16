package deps

import (
	"context"

	"github.com/google/go-licenses/v2/goutils"
	lichenmodule "github.com/google/go-licenses/v2/third_party/uw-labs/lichen/module"
	"github.com/pkg/errors"
)

type goModuleRef struct {
	// go import path, example: github.com/google/licenseclassifier/v2
	ImportPath string
	// version, example: v1.2.3, v0.0.0-20201021035429-f5854403a974
	Version string
}

// Parse dependencies from metadata in a go binary.
// Prerequisites:
// * The go binary must be built with go modules without any further modifications.
// * The command must run with working directory same as to build the analyzed
// go binary, because we need the exact go modules info used to build it.
//
// Here, I am using [1] as a short term solution. It runs [4] go version -m and parses
// output. This is preferred over [2], because [2] is an alternative implemention
// for go version -m, and I expect better long term compatibility for go version -m.
//
// The parsing command output hack is still unfavorable in the long term. As
// dicussed in [3], golang community will move go version parsing into an individual
// module in golang.org/x. I should use that module when it's there.
//
// References of similar implementations or dicussions:
// 1. https://github.com/uw-labs/lichen/blob/be9752894a5958f6ba7be9e05dc370b7a73b58db/internal/module/extract.go#L16
// 2. https://github.com/mitchellh/golicense/blob/8c09a94a11ac73299a72a68a7b41e3a737119f91/module/module.go#L27
// 3. https://github.com/golang/go/issues/39301
// 4. https://golang.org/pkg/cmd/go/internal/version/
func ListModulesInGoBinary(Path string) (refs []goModuleRef, err error) {
	defer func() {
		if err != nil {
			err = errors.Wrapf(err, "ListModulesInGoBinary(Path='%s')", Path)
		}
	}()
	depsBuildInfo, err := lichenmodule.Extract(context.Background(), Path)
	if err != nil {
		return nil, err
	}
	if len(depsBuildInfo) != 1 {
		return nil, errors.Errorf("len(depsBuildInfo) should be 1, but found %v", len(depsBuildInfo))
	}
	refs = make([]goModuleRef, 0)
	for _, buildInfo := range depsBuildInfo {
		for _, ref := range buildInfo.ModuleRefs {
			refs = append(refs, goModuleRef{
				ImportPath: ref.Path,
				Version:    ref.Version,
			})
		}
	}
	return refs, nil
}

func JoinModuleRefWithLocalModules(refs []goModuleRef) (modules []GoModule, err error) {
	localModules, err := goutils.ListModules()
	if err != nil {
		return
	}
	localModulesDict := goutils.BuildModuleDict(localModules)

	for _, ref := range refs {
		localModule, ok := localModulesDict[ref.ImportPath]
		if !ok {
			return nil, errors.Errorf("Cannot find %v in current dir's go modules. Are you running this tool from the working dir to build the binary you are analyzing?", ref.ImportPath)
		}
		if localModule.Dir == "" {
			return nil, errors.Errorf("Module %v's local directory is empty. Did you run go mod download?", ref.ImportPath)
		}
		if localModule.Version != ref.Version {
			return nil, errors.Errorf("Found %v %v in go binary, but %v is downloaded in go modules. Are you running this tool from the working dir to build the binary you are analyzing?", ref.ImportPath, ref.Version, localModule.Version)
		}
		modules = append(modules, GoModule{
			ImportPath: ref.ImportPath,
			Version:    ref.Version,
			SrcDir:     localModule.Dir,
		})
	}
	return modules, nil
}
