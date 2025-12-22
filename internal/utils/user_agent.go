package utils

import (
	"net"
	"strings"
)

// ParseUserAgent 解析 User-Agent 字符串，提取设备、操作系统和浏览器信息
func ParseUserAgent(userAgent string) (device, os, browser string) {
	if userAgent == "" {
		return "Unknown", "Unknown", "Unknown"
	}

	ua := strings.ToLower(userAgent)

	// 解析设备类型
	device = parseDevice(ua)

	// 解析操作系统
	os = parseOS(ua)

	// 解析浏览器
	browser = parseBrowser(ua)

	return device, os, browser
}

// parseDevice 解析设备类型
func parseDevice(ua string) string {
	if strings.Contains(ua, "iphone") {
		return "iPhone"
	}
	if strings.Contains(ua, "ipad") {
		return "iPad"
	}
	if strings.Contains(ua, "ipod") {
		return "iPod"
	}
	if strings.Contains(ua, "android") {
		if strings.Contains(ua, "mobile") {
			return "Android Phone"
		}
		return "Android Tablet"
	}
	if strings.Contains(ua, "windows phone") {
		return "Windows Phone"
	}
	if strings.Contains(ua, "macintosh") || strings.Contains(ua, "mac os") {
		return "Mac"
	}
	if strings.Contains(ua, "windows") {
		return "Windows PC"
	}
	if strings.Contains(ua, "linux") {
		return "Linux PC"
	}
	if strings.Contains(ua, "xbox") {
		return "Xbox"
	}
	if strings.Contains(ua, "playstation") {
		return "PlayStation"
	}
	return "Unknown"
}

// parseOS 解析操作系统
func parseOS(ua string) string {
	// iOS
	if strings.Contains(ua, "iphone os") || strings.Contains(ua, "ios") {
		// 提取 iOS 版本，例如 "iphone os 17_0" -> "iOS 17.0"
		if idx := strings.Index(ua, "iphone os "); idx != -1 {
			version := ua[idx+10:]
			if spaceIdx := strings.Index(version, " "); spaceIdx != -1 {
				version = version[:spaceIdx]
			}
			version = strings.ReplaceAll(version, "_", ".")
			return "iOS " + version
		}
		if idx := strings.Index(ua, "os "); idx != -1 && strings.Contains(ua[:idx], "iphone") {
			version := ua[idx+3:]
			if spaceIdx := strings.Index(version, " "); spaceIdx != -1 {
				version = version[:spaceIdx]
			}
			version = strings.ReplaceAll(version, "_", ".")
			return "iOS " + version
		}
		return "iOS"
	}

	// Android
	if strings.Contains(ua, "android") {
		// 提取 Android 版本，例如 "android 13" -> "Android 13"
		if idx := strings.Index(ua, "android "); idx != -1 {
			version := ua[idx+8:]
			if spaceIdx := strings.Index(version, ";"); spaceIdx != -1 {
				version = version[:spaceIdx]
			} else if spaceIdx := strings.Index(version, " "); spaceIdx != -1 {
				version = version[:spaceIdx]
			}
			return "Android " + strings.TrimSpace(version)
		}
		return "Android"
	}

	// Windows
	if strings.Contains(ua, "windows nt") {
		// Windows 版本映射
		if strings.Contains(ua, "windows nt 10.0") {
			return "Windows 10/11"
		}
		if strings.Contains(ua, "windows nt 6.3") {
			return "Windows 8.1"
		}
		if strings.Contains(ua, "windows nt 6.2") {
			return "Windows 8"
		}
		if strings.Contains(ua, "windows nt 6.1") {
			return "Windows 7"
		}
		return "Windows"
	}

	// macOS
	if strings.Contains(ua, "mac os x") || strings.Contains(ua, "macintosh") {
		// 提取 macOS 版本
		if idx := strings.Index(ua, "mac os x "); idx != -1 {
			version := ua[idx+9:]
			if spaceIdx := strings.Index(version, " "); spaceIdx != -1 {
				version = version[:spaceIdx]
			}
			version = strings.ReplaceAll(version, "_", ".")
			return "macOS " + version
		}
		return "macOS"
	}

	// Linux
	if strings.Contains(ua, "linux") {
		return "Linux"
	}

	return "Unknown"
}

// parseBrowser 解析浏览器
func parseBrowser(ua string) string {
	// Chrome
	if strings.Contains(ua, "chrome") && !strings.Contains(ua, "edg") && !strings.Contains(ua, "opr") {
		// 提取 Chrome 版本
		if idx := strings.Index(ua, "chrome/"); idx != -1 {
			version := ua[idx+7:]
			if spaceIdx := strings.Index(version, " "); spaceIdx != -1 {
				version = version[:spaceIdx]
			}
			return "Chrome " + version
		}
		return "Chrome"
	}

	// Safari
	if strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome") {
		// 提取 Safari 版本
		if idx := strings.Index(ua, "version/"); idx != -1 {
			version := ua[idx+8:]
			if spaceIdx := strings.Index(version, " "); spaceIdx != -1 {
				version = version[:spaceIdx]
			}
			return "Safari " + version
		}
		return "Safari"
	}

	// Firefox
	if strings.Contains(ua, "firefox") {
		// 提取 Firefox 版本
		if idx := strings.Index(ua, "firefox/"); idx != -1 {
			version := ua[idx+8:]
			if spaceIdx := strings.Index(version, " "); spaceIdx != -1 {
				version = version[:spaceIdx]
			}
			return "Firefox " + version
		}
		return "Firefox"
	}

	// Edge
	if strings.Contains(ua, "edg") {
		if idx := strings.Index(ua, "edg/"); idx != -1 {
			version := ua[idx+4:]
			if spaceIdx := strings.Index(version, " "); spaceIdx != -1 {
				version = version[:spaceIdx]
			}
			return "Edge " + version
		}
		return "Edge"
	}

	// Opera
	if strings.Contains(ua, "opr") {
		if idx := strings.Index(ua, "opr/"); idx != -1 {
			version := ua[idx+4:]
			if spaceIdx := strings.Index(version, " "); spaceIdx != -1 {
				version = version[:spaceIdx]
			}
			return "Opera " + version
		}
		return "Opera"
	}

	return "Unknown"
}

// GetClientIP 从 HTTP 请求中获取客户端 IP 地址
// 优先检查 X-Forwarded-For 和 X-Real-IP 头部（用于反向代理场景）
func GetClientIP(remoteAddr, forwardedFor, realIP string) string {
	// 优先使用 X-Forwarded-For（可能包含多个 IP，取第一个）
	if forwardedFor != "" {
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// 其次使用 X-Real-IP
	if realIP != "" {
		return realIP
	}

	// 最后使用 RemoteAddr
	if remoteAddr != "" {
		// RemoteAddr 格式通常是 "IP:Port"，需要提取 IP
		host, _, err := net.SplitHostPort(remoteAddr)
		if err == nil {
			return host
		}
		return remoteAddr
	}

	return "Unknown"
}
