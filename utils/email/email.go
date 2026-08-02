package email

import "gopkg.in/gomail.v2"

// ...existing code...

var (
	SMTPServer = ""
	SMTPPort   = 0
)

func SendSystemEmails(sendTo []string, subject, body string) error {
	dialer := gomail.NewDialer(SMTPServer, SMTPPort, "", "")
	m := gomail.NewMessage()
	var from = "support@grapery.xyz"
	m.SetHeader("From", from)
	m.SetHeader("To", sendTo...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)
	if err := dialer.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

// PasswordResetEmail 发送密码重置邮件
// PasswordResetEmail 发送密码重置邮件
func PasswordResetEmail(to []string, username, resetLink string) error {
	subject := "密码重置通知"
	body := `<h2>亲爱的 ` + username + `，</h2>
<p>您请求了密码重置，请点击以下链接完成操作：</p>
<p><a href="` + resetLink + `">重置密码</a></p>
<p>如果不是您本人操作，请忽略此邮件。</p>`
	return SendSystemEmails(to, subject, body)
}

// RegistrationSuccessEmail 发送注册成功通知
// RegistrationSuccessEmail 发送注册成功通知，欢迎主页
func RegistrationSuccessEmail(to []string, username string) error {
	subject := "注册成功通知"
	body := `<h2>欢迎 ` + username + ` 加入 未择</h2>
<p>您的账号已成功注册，欢迎访问 <a href='https://rankquantity.xyz'>未择主页</a>，祝您使用愉快。</p>`
	return SendSystemEmails(to, subject, body)
}

// SubscriptionActivatedEmail 用户订阅状态变更：已经订阅
// SubscriptionActivatedEmail 用户订阅状态变更：已经订阅，会员详情
func SubscriptionActivatedEmail(to []string, username, plan string, expire string) error {
	subject := "订阅激活通知"
	body := `<h2>亲爱的 ` + username + `，</h2>
<p>您的订阅已激活，感谢您的支持！</p>
<p>会员类型：` + plan + `</p>
<p>到期时间：` + expire + `</p>`
	return SendSystemEmails(to, subject, body)
}

// SubscriptionCancelledEmail 用户订阅状态变更：已经退订
// SubscriptionCancelledEmail 用户订阅状态变更：已经退订
func SubscriptionCancelledEmail(to []string, username, plan string) error {
	subject := "订阅取消通知"
	body := `<h2>亲爱的 ` + username + `，</h2>
<p>您的 ` + plan + ` 会员订阅已取消，如有疑问请联系我们。</p>`
	return SendSystemEmails(to, subject, body)
}

// SubscriptionRenewedEmail 用户订阅状态变更：已经续费
// SubscriptionRenewedEmail 用户订阅状态变更：已经续费，会员详情
func SubscriptionRenewedEmail(to []string, username, plan, expire string) error {
	subject := "订阅续费成功"
	body := `<h2>亲爱的 ` + username + `，</h2>
<p>您的 ` + plan + ` 会员已成功续费，感谢您的继续支持！</p>
<p>新到期时间：` + expire + `</p>`
	return SendSystemEmails(to, subject, body)
}

// AccountDeletedEmail 用户注销
// AccountDeletedEmail 用户注销，欢送主页
func AccountDeletedEmail(to []string, username string) error {
	subject := "账号注销通知"
	body := `<h2>亲爱的 ` + username + `，</h2>
<p>您的账号已注销，感谢您的使用。欢迎您随时访问 <a href='https://rankquantity.xyz/goodbye'>欢送主页</a>，期待未来再次见到您！</p>`
	return SendSystemEmails(to, subject, body)
}

// UserFeedbackEmail 用户反馈
// UserFeedbackEmail 用户反馈，内容描述
func UserFeedbackEmail(to []string, username, feedback string) error {
	subject := "用户反馈通知"
	body := `<h2>用户 ` + username + ` 的反馈：</h2>
<p>` + feedback + `</p>`
	return SendSystemEmails(to, subject, body)
}
