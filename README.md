Notarization Helper
===================

This tool provides a utility to wrap the [Apple notarization process][1], it supports both macOS and Linux.

## Installing

```
go install github.com/LiamHaworth/notarization-helper
```

### macOS

Usage on macOS requires that XCode be installed so that the tool can use `xcrun altool` and `xcrun stapler`.

### Linux

Usage on Linux requires that [iTMS Transporter][2] be installed and available in the system `PATH` environment variable.

## Usage

The command line utility supports to modes of operating for either notarizing a single file or multiple files.

To notarization a single file you can use the command line flags of the utility to provide the appropriate information,
for example:

```
notarization-helper --file my_installer.pkg --bundle-id "au.id.haworth.MyApplicationInstaller" --username "developer@apple.com" --password "@keychain:DeveloperPassword"
```

For notarizing multiple files you can define a JSON or YAML file that describes the information for each file to be notarized:

```yaml
username: "developer@apple.com"
password: "@keychain:DeveloperPassword"
packages:
    - file: "my_installer.pkg"
      bundle-id: "au.id.haworth.MyApplicationInstaller"
    - file: "my_application.app"
      bundle-id: "au.id.haworth.MyApplication" 
```

And then pass that file to the notarization tool:

```
notarization-helper --file definitions.yaml
```

## Stapling

If you desire the notarization helper can also staple the notarization ticket to a file so that it can be verified offline
by a device, but note this is only supported for files with the extension `.pkg`, `.dmg`, `.kext`, or `.app`.

This can be done by either supplying the `--staple` flag or specifying `staple: true`.

**NOTE:** Only `.app`, `.kext`, and `.pkg` file types are supported on Linux

## What about code signing?

If you need to code sign your package or bundle you can use the built-in `codesign` utility on macOS and on Linux you can 
use the `apple-codesign` utility from [PyOxidizer][3].

## Licence

MIT License

Copyright (c) 2019-22 Liam Haworth.

Copyright (c) 2019-22 Family Zone Cyber Safety Ltd.

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

[1]: https://developer.apple.com/documentation/security/notarizing_your_app_before_distribution
[2]: https://help.apple.com/itc/transporteruserguide/en.lproj/static.html#apdbb0ee90816044
[3]: https://github.com/indygreg/PyOxidizer/tree/main/apple-codesign