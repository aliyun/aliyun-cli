# 阿里云 CLI 自动更新检测功能总结

[文档索引](../README.md) | [English](../en/update-check.md)

CLI 默认在普通命令执行时，后台检查主程序和已安装插件的新版本，不阻塞当前命令，更新由用户手动安装。

## 自动检查与提醒

- 每 24 小时最多自动检查一次，检查失败静默处理。没有有效缓存时，本次调用不显示提醒。
- 成功命令（包括 `aliyun version`、CI、脚本和 Agent 调用）保持原有输出和退出码。
- CLI 显示普通文本错误时，可在原错误后向 stderr 追加「当前版本 → 可升级版本」和升级命令，原退出码不变。
- 支持的 AI 错误通过 `update_notices` 提供提醒，仍输出单个 JSON。插件自行输出错误并直接退出时不追加提醒。
- 同一组件的同一目标版本每 72 小时最多提醒一次；插件提醒只涉及本次调用相关的插件。

## 检查与升级

```sh
aliyun upgrade --check                    # 检查 CLI 和全部已安装插件
aliyun upgrade --check --cli-output json  # 输出带缩进的 JSON
aliyun upgrade                           # 升级 CLI
aliyun plugin update --name <插件名称>     # 升级指定插件
```

主动检查默认输出文本摘要，不安装更新。检查完成时退出码为 0，
发现新版本也一样；部分检查失败时退出码非零，并保留已取得的结果。

识别到 Homebrew 或 Linuxbrew 安装时，`aliyun upgrade` 会显示提示，并自动依次执行
`brew update` 和 `brew upgrade aliyun-cli`。检测使用官方发布源，实际安装版本以 brew 提供的版本为准。

更新检查会提示插件所需的 CLI 版本。从插件源安装或升级时，版本不足会报错并保留原插件；
执行已安装插件时，版本要求仍作为兼容性建议。

## 开关

```sh
aliyun configure update-notifier show     # 查看设置
aliyun configure update-notifier disable  # 关闭
aliyun configure update-notifier enable   # 开启
export ALIBABA_CLOUD_CLI_NO_UPDATE_NOTIFIER=1  # 通过环境变量关闭
```

设置作用于所有 Profile。环境变量禁用优先于本地设置；移除变量不会覆盖本地已关闭的设置。
