# CLI extensions

[Documentation index](../README.md) | [简体中文](../zh-CN/cliext.md)

## ossutil

Point `aliyun ossutil` at an existing ossutil v2 executable:

```sh
export ALIBABA_CLOUD_OSSUTIL_EXEC_PATH=/usr/local/bin/ossutil
```

With this variable set, `aliyun ossutil` uses the specified executable without automatic downloads, installation, or remote version checks. The environment manages the binary's version. Missing or non-executable paths fail explicitly without falling back to installation. When the variable is unset or empty, the existing automatic installation and update behavior applies.
