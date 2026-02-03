# 跨平台打包发布指南

## 概述

本文档描述 Stock Monitor 项目的跨平台打包发布流程，包括 Windows、macOS、Linux 的可执行文件和安装包的自动化构建。

## 实施完成状态

✅ **已完成的配置**：

1. ✅ `main.go` - 添加版本信息变量和 `--version` 参数支持
2. ✅ `.goreleaser.yaml` - GoReleaser 配置文件（多平台构建、打包、发布）
3. ✅ `.github/workflows/release.yaml` - GitHub Actions 自动化发布流程
4. ✅ `Makefile` - 添加 release 相关命令（检查、快照、模拟发布）
5. ✅ `.gitignore` - 忽略 GoReleaser 的 `dist/` 目录
6. ✅ `Formula/` - Homebrew formula 目录（自动管理）

---

## 发布产物

当推送新的 tag（如 `v8.0.0`）到 GitHub 后，自动生成以下产物：

| 平台 | 文件格式 | 文件名示例 |
|------|---------|-----------|
| **Windows AMD64** | `.zip` | `stock-monitor_v8.0.0_Windows_x86_64.zip` |
| **Windows ARM64** | `.zip` | `stock-monitor_v8.0.0_Windows_arm64.zip` |
| **macOS Universal** | `.dmg` | `stock-monitor_v8.0.0_macOS_universal.dmg` |
| **macOS Universal** | `.tar.gz` | `stock-monitor_v8.0.0_macOS_universal.tar.gz` |
| **Linux AMD64** | `.tar.gz` | `stock-monitor_v8.0.0_Linux_x86_64.tar.gz` |
| **Linux ARM64** | `.tar.gz` | `stock-monitor_v8.0.0_Linux_arm64.tar.gz` |
| **Debian/Ubuntu** | `.deb` | `stock-monitor_v8.0.0_linux_amd64.deb` |
| **RHEL/Fedora** | `.rpm` | `stock-monitor_v8.0.0_linux_amd64.rpm` |
| **校验和** | `.txt` | `checksums.txt` |

---

## 发布流程

### 1. 前置准备（首次发布）

在第一次发布之前，需要配置 GitHub Secrets：

1. 访问 GitHub 仓库的 **Settings** → **Secrets and variables** → **Actions**
2. 添加 Secret: `HOMEBREW_TAP_GITHUB_TOKEN`
   - 创建 Personal Access Token（经典）：https://github.com/settings/tokens
   - 权限：勾选 `repo`（完整仓库权限）
   - 将生成的 token 粘贴到 Secret 中

**注意**：`GITHUB_TOKEN` 是自动提供的，无需手动配置。

### 2. 本地测试（推荐）

在正式发布前，先在本地测试构建流程：

```bash
# 1. 安装 GoReleaser（macOS）
brew install goreleaser

# 2. 检查配置文件语法
make release-check

# 3. 本地构建快照（不发布到 GitHub）
make release-snapshot

# 4. 验证构建产物
ls -la dist/

# 5. 测试可执行文件
./dist/stock-monitor_darwin_all/stock-monitor --version
```

**预期输出**：
```
stock-monitor dev (commit: none, built: unknown)
```

### 3. 正式发布流程

确认代码无误后，执行发布：

```bash
# 1. 确保所有更改已提交
git add .
git commit -m "Release v8.0.0: Cross-platform packaging support"

# 2. 创建版本 tag（必须以 v 开头）
git tag -a v8.0.0 -m "Release v8.0.0: Cross-platform packaging and distribution"

# 3. 推送代码和 tag
git push origin master
git push origin v8.0.0
```

### 4. 自动化流程（GitHub Actions）

推送 tag 后，GitHub Actions 自动执行以下步骤：

1. **Job 1: goreleaser**（在 Ubuntu 上运行）
   - 编译所有平台的二进制文件（Windows/macOS/Linux）
   - 创建 Universal Binary（macOS Intel + Apple Silicon）
   - 打包为 `.tar.gz` 和 `.zip` 格式
   - 生成 `.deb` 和 `.rpm` 包
   - 上传到 GitHub Releases
   - 更新 Homebrew Formula
   - 上传 macOS 产物到 Artifacts

2. **Job 2: create-dmg**（在 macOS 上运行）
   - 下载 macOS 产物
   - 使用 `create-dmg` 工具创建 `.dmg` 磁盘映像
   - 上传 `.dmg` 文件到 GitHub Releases

### 5. 验证发布

访问 GitHub Releases 页面，确认：

- ✅ 所有平台的安装包已生成
- ✅ `checksums.txt` 包含所有文件的 SHA256 校验和
- ✅ Release notes 已自动生成
- ✅ Homebrew Formula 已更新（检查 `Formula/stock-monitor.rb`）

---

## 用户安装方式

### macOS

**方式 1: Homebrew（推荐）**

```bash
# 首次安装
brew tap Junrin-Lee/stock-monitor https://github.com/Junrin-Lee/stock-monitor.git
brew install stock-monitor

# 运行
stock-monitor

# 升级
brew upgrade stock-monitor
```

**方式 2: DMG 安装包**

1. 下载 `stock-monitor_v8.0.0_macOS_universal.dmg`
2. 双击打开 DMG 文件
3. 将 `stock-monitor` 拖拽到 `/Applications` 或其他目录
4. 首次运行：右键 → 打开（绕过 Gatekeeper 警告）

**方式 3: 手动安装（tar.gz）**

```bash
# 下载并解压
curl -LO https://github.com/Junrin-Lee/stock-monitor/releases/download/v8.0.0/stock-monitor_v8.0.0_macOS_universal.tar.gz
tar -xzf stock-monitor_v8.0.0_macOS_universal.tar.gz

# 移动到 PATH
sudo mv stock-monitor /usr/local/bin/

# 运行
stock-monitor --version
```

### Linux

**Debian/Ubuntu（推荐）**

```bash
# 下载 .deb 包
wget https://github.com/Junrin-Lee/stock-monitor/releases/download/v8.0.0/stock-monitor_v8.0.0_linux_amd64.deb

# 安装
sudo dpkg -i stock-monitor_v8.0.0_linux_amd64.deb

# 运行
stock-monitor

# 配置文件位置: /etc/stock-monitor/config.yaml
# 国际化文件: /usr/share/stock-monitor/i18n/
```

**RHEL/Fedora/CentOS**

```bash
# 下载 .rpm 包
wget https://github.com/Junrin-Lee/stock-monitor/releases/download/v8.0.0/stock-monitor_v8.0.0_linux_amd64.rpm

# 安装
sudo rpm -i stock-monitor_v8.0.0_linux_amd64.rpm

# 运行
stock-monitor
```

**其他发行版（tar.gz）**

```bash
# 下载并解压
wget https://github.com/Junrin-Lee/stock-monitor/releases/download/v8.0.0/stock-monitor_v8.0.0_Linux_x86_64.tar.gz
tar -xzf stock-monitor_v8.0.0_Linux_x86_64.tar.gz

# 移动到 PATH
sudo mv stock-monitor /usr/local/bin/

# 运行
stock-monitor --version
```

### Windows

**安装步骤**：

1. 下载 `stock-monitor_v8.0.0_Windows_x86_64.zip`
2. 解压到任意目录（如 `C:\Program Files\stock-monitor\`）
3. （可选）将目录添加到系统 PATH 环境变量
4. 打开 PowerShell 或 CMD，运行：

```powershell
# 导航到解压目录
cd "C:\Program Files\stock-monitor"

# 运行程序
.\stock-monitor.exe

# 查看版本
.\stock-monitor.exe --version
```

---

## 故障排查

### 问题 1: macOS 显示"无法验证开发者"

**原因**：应用未签名（开源版 GoReleaser 不支持代码签名）

**解决方案**：
1. 右键点击 `stock-monitor` → 选择"打开"
2. 或在终端执行：
   ```bash
   xattr -d com.apple.quarantine /path/to/stock-monitor
   ```

### 问题 2: GitHub Actions 发布失败 - "403 Forbidden"

**原因**：`HOMEBREW_TAP_GITHUB_TOKEN` 未配置或权限不足

**解决方案**：
1. 检查 GitHub Secrets 中是否存在 `HOMEBREW_TAP_GITHUB_TOKEN`
2. 确认 Token 权限包含 `repo`（完整仓库权限）
3. 重新生成 Token 并更新 Secret

### 问题 3: DMG 创建失败 - "create-dmg not found"

**原因**：macOS runner 未安装 `create-dmg` 工具

**解决方案**：
- GitHub Actions 工作流已包含 `brew install create-dmg` 步骤
- 如果失败，检查 Homebrew 是否正常工作
- 可暂时注释掉 `create-dmg` Job，只发布 `.tar.gz` 格式

### 问题 4: 版本信息显示为 "dev"

**原因**：未通过 GoReleaser 构建（ldflags 未注入）

**解决方案**：
- 使用 `make release-snapshot` 或 `goreleaser release --snapshot --clean` 构建
- 正式发布后，从 GitHub Releases 下载的版本会显示正确信息

### 问题 5: Linux 包安装后找不到配置文件

**检查路径**：
- Debian/Ubuntu: `/etc/stock-monitor/config.yaml`
- 国际化文件: `/usr/share/stock-monitor/i18n/`

**创建用户配置**：
```bash
# 复制示例配置到用户目录
mkdir -p ~/.config/stock-monitor
cp /etc/stock-monitor/config.yaml ~/.config/stock-monitor/config.yml
```

---

## 版本号规范

遵循语义化版本（Semantic Versioning 2.0.0）：

| 版本格式 | 示例 | 用途 |
|---------|------|------|
| `vMAJOR.MINOR.PATCH` | `v8.0.0` | 正式版本 |
| `vMAJOR.MINOR.PATCH-rc.N` | `v8.0.0-rc.1` | Release Candidate（发布候选） |
| `vMAJOR.MINOR.PATCH-beta.N` | `v8.0.0-beta.1` | Beta 测试版 |
| `vMAJOR.MINOR.PATCH-alpha.N` | `v8.0.0-alpha.1` | Alpha 内部测试版 |

**注意事项**：
- 必须以 `v` 开头（如 `v8.0.0`）
- 预发布版本会被标记为 "Pre-release" 在 GitHub Releases
- Tag 名称必须与版本号一致

---

## 清理测试发布

如果测试发布后需要清理：

```bash
# 删除 GitHub Release
gh release delete v8.0.0-rc1 --yes

# 删除本地 tag
git tag -d v8.0.0-rc1

# 删除远程 tag
git push origin :refs/tags/v8.0.0-rc1

# 清理本地构建产物
rm -rf dist/
```

---

## 未来改进

1. **代码签名**（需要付费证书）
   - macOS: Apple Developer ID 签名（避免 Gatekeeper 警告）
   - Windows: Authenticode 签名（避免 SmartScreen 警告）

2. **自动化测试**
   - 在发布前运行完整测试套件
   - 集成到 GitHub Actions workflow

3. **发布到更多平台**
   - Chocolatey（Windows 包管理器）
   - Snap Store（Linux 通用包）
   - AUR（Arch User Repository）

4. **自动更新机制**
   - 检查新版本
   - 提示用户升级

---

## 参考资源

- [GoReleaser 官方文档](https://goreleaser.com/)
- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [create-dmg 工具](https://github.com/create-dmg/create-dmg)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
- [Semantic Versioning 2.0.0](https://semver.org/)

---

**文档版本**: v1.0
**最后更新**: 2026-02-03
**状态**: ✅ 配置完成，待首次发布测试
