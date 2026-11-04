# CLI 扩展

[文档索引](../README.md) | [English](../en/cliext.md)

## ossutil

让 `aliyun ossutil` 使用已有的 ossutil v2 可执行文件：

```sh
export ALIBABA_CLOUD_OSSUTIL_EXEC_PATH=/usr/local/bin/ossutil
```

设置该变量后，`aliyun ossutil` 直接使用指定的可执行文件，跳过自动下载、安装和远端版本检查，工具版本由运行环境管理。路径不存在或不可执行时会明确报错，不回退到自动安装。未设置该变量或值为空时，保持原有自动安装和更新行为。
