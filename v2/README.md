# go-licenses v2

Find latest updates in the tracker issue: https://github.com/google/go-licenses/issues/70.

A tool to automate license management workflow for go module project's dependencies and transitive dependencies.

## Install

Download the released package and install it to your PATH, choose a release from https://github.com/Bobgy/go-licenses/releases:

```bash
curl -LO <download-url>/go-licenses-linux.tar.gz
tar xvf go-licenses-linux.tar.gz
sudo mv go-licenses/* /usr/local/bin/
# or move the content to anywhere in PATH
```

## Config & Output Examples

<!-- TODO: update NOTICES folder of this repo. -->
<!-- [NOTICES folder](./NOTICES) is an example of generated NOTICES for go-licenses tool itself. -->

Examples used in Kubeflow Pipelines:

* [go-licenses.yaml (config file)](https://github.com/kubeflow/pipelines/blob/master/v2/go-licenses.yaml)
* [licenses.csv (generated)](https://github.com/kubeflow/pipelines/blob/master/v2/third_party/licenses/launcher.csv)

## Usage

### One-off License Update

1. Get version of the repo you need licenses info:

    ```bash
    git clone <go-mod-repo-you-need-license-info>
    cd <go-mod-repo-you-need-license-info>
    git checkout <version>
    ```

1. Get dependencies from a built go binary and generate a `license_info.csv` file of their licenses:

    ```bash
    go-licenses csv <package> | tee licenses.csv
    # or
    go-licenses csv --binary <binary_path> | tee licenses.csv
    ```

    The csv file has three columns: `dependency`, `license download url` and inferred `license type`.

    Note, the format is consistent with [google/go-licenses](https://github.com/google/go-licenses).

1. The tool may fail to identify:

    * Download url of a license: they will be left out in the csv.
    * SPDX ID of a license: they will be named `Unknown` in the csv.

    Check them manually and update your `go-licenses.yaml` config to fix them, refer to [the example](./go-licenses.yaml).
    After your config fix, re-run the same command to generate licenses csv again.
    Iterate until you resolved all license issues.

1. To comply with license terms, download notices, licenses and source folders that should be distributed along with the built binary:

    ```bash
    go-licenses save <licenses_csv_path> --save_path="third_party/NOTICES"
    ```

    Notices and licenses will be concatenated to a single file `license.txt`.
    Source code folders will be copied to `<module/import/path>`.

    Some licenses will be rejected based on its [license type](https://github.com/google/licenseclassifier/blob/df6aa8a2788bdf5ac382148c2453a407a29819b8/license_type.go#L341).

### Integrating into a project with CI

What works for my project:

* Check `licenses.csv` into source control.
* During presubmit tests (alongside other go unit tests), verify `licenses.csv` is in-sync using `go-licenses csv` command.
* When building a container with the go binary (for example during release), comply to open source licenses using `go-licenses save` command.

## Implementation Details

Rough idea of steps in the two commands.

`go-licenses csv` does the following to generate the `license_info.csv`:

1. Load `go-licenses.yaml` config file, the config file can contain
    * module git branch
    * module license overrides (path excludes or directly assign result license)
1. All dependencies and transitive dependencies are listed by `go version -m <binary-path>`. When a binary is built with go modules, module info are logged inside the binary as metadata. Then we parse go CLI result to get the full list using uw-labs/lichen.
1. Scan licenses and report problems:
    1. Use <github.com/google/licenseclassifier/v2> detect licenses from all files of dependencies.
    1. Report an error if no license found for a dependency etc.
1. Get license public URLs:
    1. Get a dependency's github repo by fetching meta info like `curl 'https://k8s.io/client-go?go-get=1'`.
    1. Get dependency's version info from go modules metadata.
    1. Combine github repo, version and license file path to a public github URL to the license file.
1. Generate CSV output with module name, license URL and license type.
1. Report dependencies the tool failed to deal with during the process.

`go-licenses save` does the following:

1. Read from `license_info.csv` generated in `go-licenses csv`.
1. Call [github.com/google/licenseclassifier](https://github.com/google/licenseclassifier) to get license type.
1. Three types of reactions to license type:
    * Download its notice and license for all types.
    * Copy source folder for types that require redistribution of source code.
    * Reject according to <https://github.com/google/licenseclassifier/blob/df6aa8a2788bdf5ac382148c2453a407a29819b8/license_type.go#L341>.

## Credits

go-licenses/v2 is greatly inspired by

* [github.com/google/go-licenses](https://github.com/google/go-licenses) for the commands and compliance workflow.
* [github.com/mitchellh/golicense](https://github.com/mitchellh/golicense) for getting modules from binary.
* [github.com/uw-labs/lichen](https://github.com/uw-labs/lichen) for the configuration to facilitate incremental human validation after version upgrade.

## Comparison with similar tools

<!-- TODO(Bobgy): update this to a table -->

* go-licenses/v2 was greatly inspired by [github.com/google/go-licenses](https://github.com/google/go-licenses), with the differences:
  * go-licenses/v2 works better with go modules.
    * no need to vendor dependencies.
    * discovers versioned license URLs.
  * go-licenses/v2 scans all dependency files to find multiple licenses if any, while go-licenses detects by file name heuristics in local source folders and only finds one license per dependency.
  * go-licenses/v2 supports using a manually maintained config file `go-licenses.yaml`, so that we can reuse periodic license changes with existing information.
* go-licenses/v2 was mostly written before I learned [github.com/github/licensed](https://github.com/github/licensed) is a thing.
  * similar to google/go-licenses, github/licensed only use heuristics to find licenses and assumes one license per repo.
  * github/licensed uses a different library for detecting and classifying licenses.
* go-licenses/v2 is a rewrite of [kubeflow/testing/go-license-tools](https://github.com/kubeflow/testing/tree/master/py/kubeflow/testing/go-license-tools) in go, with many improvements:
  * better & more robust github repo resolution ratio
  * better license classification rate using google/licenseclassifier/v2 (it especially handles BSD-2-Clause and BSD-3-Clause significantly better than GitHub license API).
  * automates licenses that require distributing source code with it (copied from local module src cache)
  * simpler process e2e (instead of too many intermediate steps and config files)
  * rewritten in go, so it's easier to redistribute the binary than python

## Roadmap

General directions to improve this tool:

* Build backward compatible behavior compared to google/go-licenses v1.
* Ask for more usage & feedback and improve robustness of the tool.

## TODOs

### Features

#### P0

* [ ] Use cobra to support providing the same information via argument or config.
  * [x] BinaryPath arg.
  * [x] Output CSV to stdout.
  * [x] license_info.csv path as input arg of save command.
  * [x] save command needs --save_path flag.
* [ ] Implement "check" command.
* [x] Support use-case of one modules folder with multiple binaries.
* [x] Support customizing allowed license types.
* [x] Support replace directives.
* [x] Support modules with +incompatible in their versions, ref: <https://golang.org/ref/mod#incompatible-versions>.

#### P1

* [ ] Support installation using go get.
* [ ] Refactor & improve test coverage.

#### P2

* [ ] Support auto inclusion of licenses in headers by recording start line and end line of a license detection.
* [ ] Check header licenses match their root license.
* [ ] Find better default locations of generated files.
* [ ] Improve logging format & consistency.
* [ ] Tutorial for integration in CI/CD.

## License Workflow Design Overview

The v2 package is being developed and currently incomplete, @Bobgy is
upstreaming changes from his fork in <https://github.com/Bobgy/go-licenses/blob/main/v2>.

Tracking issue where you can find the roadmap and progress:
<https://github.com/google/go-licenses/issues/70>.

The major changes from v1 are:

* V2 only supports go modules, it can get license URL for modules without a need for you to vendor your dependencies.
* V2 does not assume each module has a single license, v2 will scan all the files for each module to find licenses.
