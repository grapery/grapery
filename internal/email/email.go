package email

import (
	"crypto/tls"
	"fmt"
	"html"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/gomail.v2"
)

var (
	SMTPServer   = ""
	SMTPPort     = 0
	SMTPUsername = ""
	SMTPPassword = ""
	SMTPFrom     = "support@grapery.xyz"
	// SMTPInsecureSkipVerify is useful for some self-hosted SMTP servers; keep false in production.
	SMTPInsecureSkipVerify = false
)

func init() {
	// Load from env at startup (works for all binaries using this package).
	if v := strings.TrimSpace(os.Getenv("SMTP_SERVER")); v != "" {
		SMTPServer = v
	}
	if v := strings.TrimSpace(os.Getenv("SMTP_PORT")); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			SMTPPort = p
		}
	}
	if v := os.Getenv("SMTP_USERNAME"); v != "" {
		SMTPUsername = v
	}
	if v := os.Getenv("SMTP_PASSWORD"); v != "" {
		SMTPPassword = v
	}
	if v := strings.TrimSpace(os.Getenv("SMTP_FROM")); v != "" {
		SMTPFrom = v
	}
	if v := strings.TrimSpace(os.Getenv("SMTP_INSECURE_SKIP_VERIFY")); v != "" {
		SMTPInsecureSkipVerify = v == "1" || strings.EqualFold(v, "true")
	}
}

// baseEmailTemplate 基础邮件模板，统一样式
func baseEmailTemplate(title, content, footer string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0; background-color: #f5f5f5; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .email-wrapper { background-color: #ffffff; border-radius: 12px; box-shadow: 0 2px 8px rgba(0,0,0,0.08); overflow: hidden; }
        .header { background: linear-gradient(135deg, #6366f1 0%%, #8b5cf6 100%%); padding: 32px 24px; text-align: center; }
        .header h1 { color: #ffffff; margin: 0; font-size: 24px; font-weight: 600; }
        .header .logo { font-size: 28px; margin-bottom: 8px; }
        .content { padding: 32px 24px; }
        .content h2 { color: #1f2937; margin-top: 0; font-size: 20px; }
        .content p { color: #4b5563; margin: 16px 0; }
        .button { display: inline-block; background: linear-gradient(135deg, #6366f1 0%%, #8b5cf6 100%%); color: #ffffff !important; text-decoration: none; padding: 14px 32px; border-radius: 8px; font-weight: 600; margin: 16px 0; }
        .button:hover { opacity: 0.9; }
        .info-box { background-color: #f3f4f6; border-radius: 8px; padding: 16px; margin: 16px 0; }
        .info-box p { margin: 8px 0; color: #374151; }
        .info-label { font-weight: 600; color: #6366f1; }
        .footer { background-color: #f9fafb; padding: 24px; text-align: center; border-top: 1px solid #e5e7eb; }
        .footer p { color: #9ca3af; font-size: 12px; margin: 4px 0; }
        .footer a { color: #6366f1; text-decoration: none; }
        .warning { background-color: #fef3c7; border-left: 4px solid #f59e0b; padding: 12px 16px; margin: 16px 0; border-radius: 0 8px 8px 0; }
        .warning p { color: #92400e; margin: 0; }
        .success { background-color: #d1fae5; border-left: 4px solid #10b981; padding: 12px 16px; margin: 16px 0; border-radius: 0 8px 8px 0; }
        .success p { color: #065f46; margin: 0; }
        .code-box { background-color: #1f2937; color: #ffffff; padding: 20px; border-radius: 8px; text-align: center; font-size: 32px; letter-spacing: 8px; font-weight: bold; margin: 16px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="email-wrapper">
            <div class="header">
                <div class="logo">📖</div>
                <h1>未择</h1>
            </div>
            <div class="content">
                %s
            </div>
            <div class="footer">
                %s
                <p>© 2024 未择. 保留所有权利。</p>
                <p><a href="https://rankquantity.xyz">访问官网</a> | <a href="https://rankquantity.xyz/privacy">隐私政策</a> | <a href="https://rankquantity.xyz/terms">服务条款</a></p>
            </div>
        </div>
    </div>
</body>
</html>`, title, content, footer)
}

// SendSystemEmails 发送系统邮件
func SendSystemEmails(sendTo []string, subject, body string) error {
	if strings.TrimSpace(SMTPServer) == "" || SMTPPort == 0 {
		return fmt.Errorf("smtp not configured")
	}
	dialer := gomail.NewDialer(SMTPServer, SMTPPort, SMTPUsername, SMTPPassword)
	// Common defaults: port 465 is implicit TLS.
	if SMTPPort == 465 {
		dialer.SSL = true
	}
	dialer.TLSConfig = &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: SMTPInsecureSkipVerify,
	}
	m := gomail.NewMessage()
	m.SetHeader("From", SMTPFrom)
	m.SetHeader("To", sendTo...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	if err := dialer.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

// PasswordResetEmail 发送密码重置邮件
func PasswordResetEmail(to []string, username, resetLink string) error {
	subject := "【未择】密码重置"
	content := fmt.Sprintf(`
        <h2>亲爱的 %s，</h2>
        <p>我们收到了您的密码重置请求。请点击下方按钮完成密码重置：</p>
        <p style="text-align: center;">
            <a href="%s" class="button">重置密码</a>
        </p>
        <div class="warning">
            <p>⚠️ 此链接将在 15 分钟后失效。如果不是您本人操作，请忽略此邮件并确保您的账户安全。</p>
        </div>
        <p>如果按钮无法点击，请复制以下链接到浏览器：</p>
        <p style="word-break: break-all; color: #6366f1;">%s</p>
    `, html.EscapeString(username), resetLink, resetLink)
	footer := `<p>如有任何疑问，请联系 <a href="mailto:support@grapery.xyz">support@grapery.xyz</a></p>`
	body := baseEmailTemplate(subject, content, footer)
	return SendSystemEmails(to, subject, body)
}

// RegistrationSuccessEmail 发送注册成功通知
func RegistrationSuccessEmail(to []string, username string) error {
	subject := "【未择】欢迎加入"
	content := fmt.Sprintf(`
        <h2>欢迎 %s 加入未择！ 🎉</h2>
        <div class="success">
            <p>✅ 您的账号已成功注册！</p>
        </div>
        <p>未择是一个创意写作与故事分享平台，在这里您可以：</p>
        <ul style="color: #4b5563;">
            <li>创作和分享您的原创故事</li>
            <li>与志同道合的创作者互动交流</li>
            <li>发现精彩的故事内容</li>
            <li>参与社区活动和创作挑战</li>
        </ul>
        <p style="text-align: center;">
            <a href="https://rankquantity.xyz" class="button">开始探索</a>
        </p>
    `, html.EscapeString(username))
	footer := `<p>期待在未择与您相遇！</p>`
	body := baseEmailTemplate(subject, content, footer)
	return SendSystemEmails(to, subject, body)
}

// SubscriptionActivatedEmail 用户订阅状态变更：已经订阅
func SubscriptionActivatedEmail(to []string, username, plan string, expire string) error {
	subject := "【未择】会员激活成功"
	content := fmt.Sprintf(`
        <h2>亲爱的 %s，</h2>
        <div class="success">
            <p>🎊 恭喜！您的会员订阅已成功激活！</p>
        </div>
        <p>感谢您对未择的支持，您现在可以享受以下专属权益：</p>
        <div class="info-box">
            <p><span class="info-label">会员类型：</span>%s</p>
            <p><span class="info-label">到期时间：</span>%s</p>
        </div>
        <p>会员专属权益包括：</p>
        <ul style="color: #4b5563;">
            <li>无限制故事创作与存储</li>
            <li>高级 AI 创作助手</li>
            <li>专属会员标识</li>
            <li>优先客服支持</li>
        </ul>
        <p style="text-align: center;">
            <a href="https://rankquantity.xyz/membership" class="button">查看会员中心</a>
        </p>
    `, html.EscapeString(username), html.EscapeString(plan), html.EscapeString(expire))
	footer := `<p>如有任何疑问，请联系 <a href="mailto:support@grapery.xyz">support@grapery.xyz</a></p>`
	body := baseEmailTemplate(subject, content, footer)
	return SendSystemEmails(to, subject, body)
}

// SubscriptionCancelledEmail 用户订阅状态变更：已经退订
func SubscriptionCancelledEmail(to []string, username, plan string) error {
	subject := "【未择】订阅取消确认"
	content := fmt.Sprintf(`
        <h2>亲爱的 %s，</h2>
        <p>您的 <strong>%s</strong> 会员订阅已成功取消。</p>
        <div class="info-box">
            <p>在当前订阅期结束前，您仍可继续使用所有会员功能。</p>
        </div>
        <p>我们非常珍视您在未择的时光。如果您是因为遇到问题而取消订阅，欢迎随时告诉我们，我们会尽力改进。</p>
        <p>期待未来能再次为您服务！</p>
        <p style="text-align: center;">
            <a href="https://rankquantity.xyz/membership" class="button">重新订阅</a>
        </p>
    `, html.EscapeString(username), html.EscapeString(plan))
	footer := `<p>如需帮助，请联系 <a href="mailto:support@grapery.xyz">support@grapery.xyz</a></p>`
	body := baseEmailTemplate(subject, content, footer)
	return SendSystemEmails(to, subject, body)
}

// SubscriptionRenewedEmail 用户订阅状态变更：已经续费
func SubscriptionRenewedEmail(to []string, username, plan, expire string) error {
	subject := "【未择】订阅续费成功"
	content := fmt.Sprintf(`
        <h2>亲爱的 %s，</h2>
        <div class="success">
            <p>✅ 您的 %s 会员已成功续费！</p>
        </div>
        <p>感谢您的持续支持与信任！</p>
        <div class="info-box">
            <p><span class="info-label">会员类型：</span>%s</p>
            <p><span class="info-label">新到期时间：</span>%s</p>
        </div>
        <p style="text-align: center;">
            <a href="https://rankquantity.xyz/membership" class="button">查看会员中心</a>
        </p>
    `, html.EscapeString(username), html.EscapeString(plan), html.EscapeString(plan), html.EscapeString(expire))
	footer := `<p>如有任何疑问，请联系 <a href="mailto:support@grapery.xyz">support@grapery.xyz</a></p>`
	body := baseEmailTemplate(subject, content, footer)
	return SendSystemEmails(to, subject, body)
}

// AccountDeletionGracePeriodScheduledEmail notifies the user that account closure enters the grace/review window.
func AccountDeletionGracePeriodScheduledEmail(to []string, username string, scheduledUnix int64) error {
	t := time.Unix(scheduledUnix, 0).UTC().Format("2006-01-02 15:04 (UTC)")
	subject := "【未择】账号注销已进入冷静期"
	content := fmt.Sprintf(`
        <h2>亲爱的 %s，</h2>
        <p>我们已收到您的账号注销申请。</p>
        <div class="warning">
            <p>⚠️ 冷静期内，您在应用内仍可登录并可撤销注销。<b>非公开内容与草稿将在正式注销时被删除</b>；公开的已发布故事/故事板等将匿名保留。</p>
        </div>
        <div class="info-box">
            <p><span class="info-label">预定处理完成时间：</span>%s</p>
        </div>
        <p style="text-align: center;">
            <a href="https://rankquantity.xyz" class="button">打开未择客户端</a>
        </p>
    `, html.EscapeString(username), html.EscapeString(t))
	footer := `<p>如有疑问，请联系 <a href="mailto:support@grapery.xyz">support@grapery.xyz</a></p>`
	body := baseEmailTemplate(subject, content, footer)
	return SendSystemEmails(to, subject, body)
}

// AccountDeletedEmail 用户注销
func AccountDeletedEmail(to []string, username string) error {
	subject := "【未择】账号注销确认"
	content := fmt.Sprintf(`
        <h2>亲爱的 %s，</h2>
        <p>您的未择账号已成功注销。</p>
        <p>感谢您曾经与我们一起创作和分享故事的时光。您的创作足迹将被我们铭记。</p>
        <div class="info-box">
            <p>您的账户数据已按照隐私政策进行处理。如需了解更多信息，请查阅我们的隐私政策。</p>
        </div>
        <p>如果您将来想要回归，我们随时欢迎您重新注册！</p>
        <p style="text-align: center;">
            <a href="https://rankquantity.xyz" class="button">再见，期待重逢</a>
        </p>
    `, html.EscapeString(username))
	footer := `<p>祝您一切顺利！</p>`
	body := baseEmailTemplate(subject, content, footer)
	return SendSystemEmails(to, subject, body)
}

// UserFeedbackEmail 用户反馈
func UserFeedbackEmail(to []string, username, feedback string) error {
	subject := "【未择】用户反馈"
	content := fmt.Sprintf(`
        <h2>新的用户反馈</h2>
        <div class="info-box">
            <p><span class="info-label">用户：</span>%s</p>
        </div>
        <p><strong>反馈内容：</strong></p>
        <div style="background-color: #f3f4f6; border-radius: 8px; padding: 16px; margin: 16px 0; white-space: pre-wrap;">%s</div>
    `, html.EscapeString(username), html.EscapeString(feedback))
	footer := `<p>此邮件由系统自动发送</p>`
	body := baseEmailTemplate(subject, content, footer)
	return SendSystemEmails(to, subject, body)
}

// VerificationCodeEmail 发送验证码邮件
func VerificationCodeEmail(to []string, username, code string, expireMinutes int) error {
	subject := "【未择】验证码"
	content := fmt.Sprintf(`
        <h2>亲爱的 %s，</h2>
        <p>您的验证码是：</p>
        <div class="code-box">%s</div>
        <div class="warning">
            <p>⚠️ 验证码将在 %d 分钟后失效。请勿将验证码透露给他人。</p>
        </div>
        <p>如果这不是您的操作，请忽略此邮件并检查您的账户安全。</p>
    `, html.EscapeString(username), html.EscapeString(code), expireMinutes)
	footer := `<p>如有任何疑问，请联系 <a href="mailto:support@grapery.xyz">support@grapery.xyz</a></p>`
	body := baseEmailTemplate(subject, content, footer)
	return SendSystemEmails(to, subject, body)
}

// SecurityAlertEmail 安全警报邮件
func SecurityAlertEmail(to []string, username, alertType, details, ipAddress, location string) error {
	subject := "【未择】安全提醒"
	content := fmt.Sprintf(`
        <h2>亲爱的 %s，</h2>
        <div class="warning">
            <p>🔔 检测到您的账户有新的安全活动</p>
        </div>
        <div class="info-box">
            <p><span class="info-label">活动类型：</span>%s</p>
            <p><span class="info-label">详情：</span>%s</p>
            <p><span class="info-label">IP 地址：</span>%s</p>
            <p><span class="info-label">位置：</span>%s</p>
        </div>
        <p>如果这是您本人的操作，可以忽略此邮件。</p>
        <p>如果不是您本人操作，请立即修改密码并检查账户安全。</p>
        <p style="text-align: center;">
            <a href="https://rankquantity.xyz/settings/security" class="button">检查账户安全</a>
        </p>
    `, html.EscapeString(username), html.EscapeString(alertType), html.EscapeString(details),
		html.EscapeString(ipAddress), html.EscapeString(location))
	footer := `<p>如有任何疑问，请联系 <a href="mailto:support@grapery.xyz">support@grapery.xyz</a></p>`
	body := baseEmailTemplate(subject, content, footer)
	return SendSystemEmails(to, subject, body)
}

// PaymentSuccessEmail 支付成功邮件
func PaymentSuccessEmail(to []string, username, orderID, amount, productName string) error {
	subject := "【未择】支付成功"
	content := fmt.Sprintf(`
        <h2>亲爱的 %s，</h2>
        <div class="success">
            <p>✅ 您的支付已成功！</p>
        </div>
        <div class="info-box">
            <p><span class="info-label">订单号：</span>%s</p>
            <p><span class="info-label">商品：</span>%s</p>
            <p><span class="info-label">金额：</span>¥%s</p>
        </div>
        <p>感谢您的购买！如需查看订单详情或发票，请访问订单中心。</p>
        <p style="text-align: center;">
            <a href="https://rankquantity.xyz/orders" class="button">查看订单</a>
        </p>
    `, html.EscapeString(username), html.EscapeString(orderID),
		html.EscapeString(productName), html.EscapeString(amount))
	footer := `<p>如有任何疑问，请联系 <a href="mailto:support@grapery.xyz">support@grapery.xyz</a></p>`
	body := baseEmailTemplate(subject, content, footer)
	return SendSystemEmails(to, subject, body)
}

// PaymentFailedEmail 支付失败邮件
func PaymentFailedEmail(to []string, username, orderID, amount, reason string) error {
	subject := "【未择】支付失败"
	content := fmt.Sprintf(`
        <h2>亲爱的 %s，</h2>
        <div class="warning">
            <p>⚠️ 您的支付未能完成</p>
        </div>
        <div class="info-box">
            <p><span class="info-label">订单号：</span>%s</p>
            <p><span class="info-label">金额：</span>¥%s</p>
            <p><span class="info-label">失败原因：</span>%s</p>
        </div>
        <p>请检查您的支付方式并重试。如果问题持续存在，请联系客服。</p>
        <p style="text-align: center;">
            <a href="https://rankquantity.xyz/orders/%s" class="button">重新支付</a>
        </p>
    `, html.EscapeString(username), html.EscapeString(orderID),
		html.EscapeString(amount), html.EscapeString(reason), html.EscapeString(orderID))
	footer := `<p>如需帮助，请联系 <a href="mailto:support@grapery.xyz">support@grapery.xyz</a></p>`
	body := baseEmailTemplate(subject, content, footer)
	return SendSystemEmails(to, subject, body)
}

// StoryPublishedEmail 故事发布通知
func StoryPublishedEmail(to []string, username, storyTitle, storyLink string) error {
	subject := "【未择】故事发布成功"
	content := fmt.Sprintf(`
        <h2>亲爱的 %s，</h2>
        <div class="success">
            <p>🎉 恭喜！您的故事已成功发布！</p>
        </div>
        <div class="info-box">
            <p><span class="info-label">故事标题：</span>%s</p>
        </div>
        <p>您的故事现已对所有用户可见。分享您的创作，让更多人看到您的精彩故事！</p>
        <p style="text-align: center;">
            <a href="%s" class="button">查看故事</a>
        </p>
    `, html.EscapeString(username), html.EscapeString(storyTitle), storyLink)
	footer := `<p>继续创作，分享更多精彩！</p>`
	body := baseEmailTemplate(subject, content, footer)
	return SendSystemEmails(to, subject, body)
}

// NewFollowerEmail 新粉丝通知
func NewFollowerEmail(to []string, username, followerName, followerLink string) error {
	subject := "【未择】有人关注了你"
	content := fmt.Sprintf(`
        <h2>亲爱的 %s，</h2>
        <p>🎊 好消息！<strong>%s</strong> 开始关注你了！</p>
        <p>点击下方按钮查看 TA 的主页：</p>
        <p style="text-align: center;">
            <a href="%s" class="button">查看主页</a>
        </p>
    `, html.EscapeString(username), html.EscapeString(followerName), followerLink)
	footer := `<p>继续创作优质内容，吸引更多粉丝！</p>`
	body := baseEmailTemplate(subject, content, footer)
	return SendSystemEmails(to, subject, body)
}

// NewCommentEmail 新评论通知
func NewCommentEmail(to []string, username, commenterName, storyTitle, comment, storyLink string) error {
	subject := "【未择】你的故事收到了新评论"
	content := fmt.Sprintf(`
        <h2>亲爱的 %s，</h2>
        <p><strong>%s</strong> 评论了你的故事《%s》：</p>
        <div style="background-color: #f3f4f6; border-left: 4px solid #6366f1; padding: 16px; margin: 16px 0; border-radius: 0 8px 8px 0;">
            <p style="margin: 0; color: #374151; font-style: italic;">"%s"</p>
        </div>
        <p style="text-align: center;">
            <a href="%s" class="button">查看评论</a>
        </p>
    `, html.EscapeString(username), html.EscapeString(commenterName),
		html.EscapeString(storyTitle), html.EscapeString(comment), storyLink)
	footer := `<p>与读者互动，建立更紧密的联系！</p>`
	body := baseEmailTemplate(subject, content, footer)
	return SendSystemEmails(to, subject, body)
}

// WelcomeBackEmail 回归欢迎邮件
func WelcomeBackEmail(to []string, username string, daysAway int) error {
	subject := "【未择】好久不见"
	content := fmt.Sprintf(`
        <h2>亲爱的 %s，好久不见！ 👋</h2>
        <p>我们注意到您已经有 %d 天没有访问未择了，我们非常想念您！</p>
        <p>在您离开的这段时间，平台有了很多精彩的新内容和新功能等您来探索。</p>
        <div class="info-box">
            <p>🌟 新增了更强大的 AI 创作助手</p>
            <p>📚 社区中涌现了许多优质故事</p>
            <p>👥 更多创作者加入了我们</p>
        </div>
        <p style="text-align: center;">
            <a href="https://rankquantity.xyz" class="button">立即回归</a>
        </p>
    `, html.EscapeString(username), daysAway)
	footer := `<p>期待与您再次相遇！</p>`
	body := baseEmailTemplate(subject, content, footer)
	return SendSystemEmails(to, subject, body)
}
