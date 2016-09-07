# PCF Dev cf CLI Plugin

This repository contains the source code for the PCF Dev cf CLI plugin.

Please open issues at the [PCF Dev Github issues page](https://github.com/pivotal-cf/pcfdev/issues).

## Custom OVAs

To start a custom OVA with the CLI plugin, you may use the `-o` flag:
```
$ cf dev start -o custom.ova
```
This will disable various checks for system requirements such as system memory.

## Building

The `bin/build` script will compile a version of the plugin that downloads the latest release of the PCF Dev OVA.

## Tests

Test scripts live in the `bin` directory. You must have a PivNet API token to run the tests.
