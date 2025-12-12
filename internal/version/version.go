package version

import (
	"fmt"
	"runtime"
)

var (
	// Version 版本号
	Version = "1.0.0"
	// GitCommit Git提交哈希
	GitCommit = "unknown"
	// BuildDate 构建日期
	BuildDate = "unknown"
	// GoVersion Go版本
	GoVersion = runtime.Version()
)

// GetVersion 获取版本号
func GetVersion() string {
	return Version
}

// GetFullVersion 获取完整版本信息
func GetFullVersion() string {
	return fmt.Sprintf("%s-%s", Version, GitCommit[:7])
}

// PrintFullVersionInfo 打印完整版本信息
func PrintFullVersionInfo() {
	fmt.Printf("Version:    %s\n", Version)
	fmt.Printf("Git Commit: %s\n", GitCommit)
	fmt.Printf("Build Date: %s\n", BuildDate)
	fmt.Printf("Go Version: %s\n", GoVersion)
	fmt.Printf("OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
