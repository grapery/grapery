package thirdpart

import (
	"fmt"
	"net/http"
)

func main() {
	// 创建 Google 登录实例
	googleLogin := NewGoogleLogin(
		"your-client-id",
		"your-client-secret",
		"http://your-domain/callback",
	)

	// 初始化配置
	googleLogin.Init()

	// 在你的 HTTP 处理器中：

	// 1. 登录页面处理器
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		// 生成随机状态值（防止 CSRF 攻击）
		state := "random-state-string"

		// 获取授权 URL
		authURL := googleLogin.GetAuthURL(state)

		// 重定向到 Google 登录页面
		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	})

	// 2. 回调处理器
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		// 验证状态值
		if state != "your-saved-state" {
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}

		// 交换授权码获取令牌
		token, err := googleLogin.Exchange(r.Context(), code)
		if err != nil {
			http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
			return
		}

		// 获取用户信息
		userInfo, err := googleLogin.GetUserInfo(r.Context(), token)
		if err != nil {
			http.Error(w, "Failed to get user info", http.StatusInternalServerError)
			return
		}

		// 使用 Gmail 客户端
		gmailClient := googleLogin.GetGmailClient(r.Context(), token)

		// 保存用户信息和令牌
		// ... 存储到数据库等
		_ = gmailClient
		// 返回成功响应
		fmt.Fprintf(w, "Welcome %s!", userInfo.Name)
	})
}
