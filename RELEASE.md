# 发布说明 / Release Guide

本 Fork 提供带有原生 Chain 功能的 sing-box 可执行文件。

## 如何获取发布包

### 方式一：使用 GitHub Releases（推荐）

1. 在本仓库创建并推送一个带有 `chain` 标识的 tag，例如：

```bash
git checkout chain-dev
git pull
git tag v1.12.0-chain.1
git push origin v1.12.0-chain.1
```

2. 推送 tag 后，GitHub Actions 会自动构建多平台二进制并创建 Release。

3. 到 [Releases 页面](https://github.com/dukangalex/sing-box/releases) 下载对应平台的压缩包。

### 方式二：手动触发

在 GitHub 仓库的 **Actions** → **Release Chain Fork** → **Run workflow**，输入想要的 tag 名称即可。

### 支持的平台

- Linux amd64 / arm64
- Windows amd64
- macOS amd64 / arm64
- Android arm64

## 使用

下载后解压得到 `sing-box` 可执行文件，与官方使用方式完全一致，额外支持：

```json
{
  "type": "chain",
  "tag": "my-chain",
  "outbounds": ["entry-proxy", "exit-proxy"]
}
```

详细文档见 [docs/CHAIN.md](docs/CHAIN.md)。

## 版本建议

建议使用 `v官方版本-chain.序号` 的形式，例如：

- `v1.12.0-chain.1`
- `v1.13.0-chain.1`

便于后续与官方版本对应和同步。
