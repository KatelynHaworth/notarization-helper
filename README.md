Notarization Helper
===================

This tool provides the ability to both [code sign][1] (soon™️) and [notarize][2] macOS applications on any platform, not just on
macOS itself, to make the development of macOS applications on other operating systems just a bit easier.

## Installing

Installing `notarization-helper` is as simple as using `go install`, once installed you can confirm the version installed
on your system by running `notarization-helper --version`.

```
go install github.com/KatelynHaworth/notarization-helper/v2@latest
```

**NOTE:** On macOS systems installation requires CGo as part of keychain support, to disable this install with the `disable_keychain`
tag set.

## Configuration

As this utility is designed to allow the code signing and notarization of multiple files at the same time a configuration
file is required to inform the utility of both identity and the packages to sign/notarize.

By default, the utility will look for a file called `notarization.yaml` in the same directory as where the command is
invoked but, this can be overridden using the `-f` flag to point to a specific configuration file.

The configuration itself is allowed to be either YAML or JSON and the utility will automatically detect the type based
on the file extension, `.yaml`/`.yml` or `.json` respectively.

```yaml
config_version: 2 # Required to let utility determine the config format

notary_auth:
  key_id:     "2X9R4HXF34"                           # Identifier of the App Store Connect API key
  key_file:   "app_store_connect.key"                # Path to the App Store Connect API key.
  key_issuer: "57246542-96fe-1a63-e053-0824d011072a" # Identifier of the App Store Connect team that issued the key (optional)

packages:
  - file:      "my_cool_app.app"        # Path to the package to sign and/or notarize
    bundle_id: "com.mycompany.cool_app" # Identifier for the package, only required for code signing
    staple:    true                     # Should the notarization ticket be stapled to the package
```

To support usage of this tool in a containerized environment as part of a CI/CD pipeline the following fields can be
loaded from environment variables:

  * `key_file` - Specify the environment variable by setting the value to `ENV:my_env_var`, the value of the variable must
                 be base64 encoded.

### Backward compatability

To ensure backwards compatability with version 1 of this utility, v2 can be running using legacy command line flags or 
legacy configuration files.

This support only extends to the notarization sub-command, but as such if the utility is invoked with no sub-command supplied
(e.g. `notarization-helper notarize`) it will default to the notarization sub-command.

### Stapling

If you desire the notarization helper can also staple the notarization ticket to a file so that it can be verified offline
by a device, but note this is only supported for files the following file types:

  * `.pkg` - Installer packages
  * `.dmg` - Disk image files
  * `.app`, `.kext` - macOS bundles

## Usage

### Code Signing (Coming Soon ™️)

If you need to code sign your package or bundle you can use the built-in `codesign` utility on macOS and on Linux you can
use the `apple-codesign` utility from [PyOxidizer][3].

### Notarize

*Usage:* `notarization-helper notarize`

When invoked, this command will internally launch a set of workers for each package defined in the utility configuration,
each worker handles uploading, waiting for, and stapling steps of notarization for the package the worker is assigned too.

Upon successful completion each worker will write a notarization log to a file next to the package containing the output
from the Notary API.

If the log returned by the Notary API includes one or more issues the utility will print a warning-level log message for
the package the notarization log is associated to.

```text
2025-04-19T00:30:00+10:00 WRN This package has one or more issues detected by the Notary file=my_cool_app.app numIssues=1 submissionId=00000000-85b1-4e65-afed-dcfe9b5c6fce 
```

## Licence

MIT License

Copyright (c) 2019-25 Katelyn Haworth.

Copyright (c) 2019-25 Qoria Ltd.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

[1]: https://developer.apple.com/documentation/security/code-signing-services
[2]: https://developer.apple.com/documentation/security/notarizing_your_app_before_distribution
[3]: https://github.com/indygreg/PyOxidizer/tree/main/apple-codesign