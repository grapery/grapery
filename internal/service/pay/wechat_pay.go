package pay

import (
	"context"
	"crypto/rsa"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

// WechatPayConfig holds WeChat Pay merchant credentials for Native (QR) checkout.
type WechatPayConfig struct {
	AppID     string
	MchID     string
	SerialNo  string
	APIv3Key  string
	NotifyURL string
	// PrivateKeyPEM is raw PEM; PrivateKeyPath loads from file when PEM empty.
	PrivateKeyPEM  string
	PrivateKeyPath string
}

func wechatPayConfigFromEnv() *WechatPayConfig {
	appID := strings.TrimSpace(firstNonEmpty(os.Getenv("WECHAT_PAY_APP_ID"), os.Getenv("WECHAT_APP_ID")))
	return &WechatPayConfig{
		AppID:          appID,
		MchID:          strings.TrimSpace(os.Getenv("WECHAT_PAY_MCH_ID")),
		SerialNo:       strings.TrimSpace(os.Getenv("WECHAT_PAY_SERIAL_NO")),
		APIv3Key:       strings.TrimSpace(os.Getenv("WECHAT_PAY_API_V3_KEY")),
		NotifyURL:      strings.TrimSpace(os.Getenv("WECHAT_PAY_NOTIFY_URL")),
		PrivateKeyPEM:  normalizePEM(os.Getenv("WECHAT_PAY_PRIVATE_KEY")),
		PrivateKeyPath: strings.TrimSpace(os.Getenv("WECHAT_PAY_PRIVATE_KEY_PATH")),
	}
}

func (c *WechatPayConfig) configured() bool {
	if c == nil {
		return false
	}
	hasKey := c.PrivateKeyPEM != "" || c.PrivateKeyPath != ""
	return c.AppID != "" && c.MchID != "" && c.SerialNo != "" && c.APIv3Key != "" && c.NotifyURL != "" && hasKey
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func normalizePEM(pem string) string {
	pem = strings.TrimSpace(pem)
	if pem == "" {
		return ""
	}
	return strings.ReplaceAll(pem, `\n`, "\n")
}

type wechatPayRuntime struct {
	cfg       *WechatPayConfig
	client    *core.Client
	handler   *notify.Handler
	privKey   *rsa.PrivateKey
	initOnce  sync.Once
	initErr   error
	logger    *logrus.Logger
}

func newWechatPayRuntime(logger *logrus.Logger, cfg *WechatPayConfig) *wechatPayRuntime {
	if cfg == nil {
		cfg = wechatPayConfigFromEnv()
	}
	if logger == nil {
		logger = logrus.StandardLogger()
	}
	return &wechatPayRuntime{cfg: cfg, logger: logger}
}

func (r *wechatPayRuntime) ensure() error {
	r.initOnce.Do(func() {
		if !r.cfg.configured() {
			r.initErr = fmt.Errorf("WeChat Pay is not configured (need WECHAT_PAY_MCH_ID, WECHAT_PAY_SERIAL_NO, WECHAT_PAY_API_V3_KEY, WECHAT_PAY_NOTIFY_URL, private key, and WECHAT_PAY_APP_ID/WECHAT_APP_ID)")
			return
		}
		var (
			priv *rsa.PrivateKey
			err  error
		)
		if r.cfg.PrivateKeyPEM != "" {
			priv, err = utils.LoadPrivateKey(r.cfg.PrivateKeyPEM)
		} else {
			priv, err = utils.LoadPrivateKeyWithPath(r.cfg.PrivateKeyPath)
		}
		if err != nil {
			r.initErr = fmt.Errorf("load WeChat Pay private key: %w", err)
			return
		}
		r.privKey = priv

		ctx := context.Background()
		opts := []core.ClientOption{
			option.WithWechatPayAutoAuthCipher(r.cfg.MchID, r.cfg.SerialNo, priv, r.cfg.APIv3Key),
		}
		client, err := core.NewClient(ctx, opts...)
		if err != nil {
			r.initErr = fmt.Errorf("init WeChat Pay client: %w", err)
			return
		}
		r.client = client

		// Certificate visitor for notify verification (auto-downloaded platform certs).
		certVisitor := downloader.MgrInstance().GetCertificateVisitor(r.cfg.MchID)
		handler, err := notify.NewRSANotifyHandler(r.cfg.APIv3Key, verifiers.NewSHA256WithRSAVerifier(certVisitor))
		if err != nil {
			r.initErr = fmt.Errorf("init WeChat Pay notify handler: %w", err)
			return
		}
		r.handler = handler
	})
	return r.initErr
}

// CreateNativeOrder creates a Native (QR) prepay and returns code_url + out_trade_no.
func (r *wechatPayRuntime) CreateNativeOrder(ctx context.Context, outTradeNo, description string, amountFen int64) (codeURL string, err error) {
	if err := r.ensure(); err != nil {
		return "", err
	}
	if amountFen <= 0 {
		return "", fmt.Errorf("invalid WeChat Pay amount")
	}
	// WeChat Native settles in CNY fen.
	svc := native.NativeApiService{Client: r.client}
	resp, result, err := svc.Prepay(ctx, native.PrepayRequest{
		Appid:       core.String(r.cfg.AppID),
		Mchid:       core.String(r.cfg.MchID),
		Description: core.String(truncateRunes(description, 127)),
		OutTradeNo:  core.String(outTradeNo),
		NotifyUrl:   core.String(r.cfg.NotifyURL),
		Amount: &native.Amount{
			Total:    core.Int64(amountFen),
			Currency: core.String("CNY"),
		},
	})
	if err != nil {
		status := 0
		if result != nil && result.Response != nil {
			status = result.Response.StatusCode
		}
		return "", fmt.Errorf("WeChat Native prepay failed (status=%d): %w", status, err)
	}
	if resp == nil || resp.CodeUrl == nil || *resp.CodeUrl == "" {
		return "", fmt.Errorf("WeChat Native prepay returned empty code_url")
	}
	return *resp.CodeUrl, nil
}

// ParseNotify verifies and decrypts a WeChat Pay notification.
func (r *wechatPayRuntime) ParseNotify(ctx context.Context, request *http.Request) (*notify.Request, *payments.Transaction, error) {
	if err := r.ensure(); err != nil {
		return nil, nil, err
	}
	transaction := new(payments.Transaction)
	notifyReq, err := r.handler.ParseNotifyRequest(ctx, request, transaction)
	if err != nil {
		return nil, nil, err
	}
	return notifyReq, transaction, nil
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}

func wechatOutTradeNoFromPaymentID(paymentID string) string {
	// WeChat out_trade_no: 6–32 chars, digits/letters underscore hyphen.
	id := strings.ReplaceAll(paymentID, "pay_", "w")
	id = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			return r
		default:
			return -1
		}
	}, id)
	if len(id) < 6 {
		id = fmt.Sprintf("w%x", time.Now().UnixNano())
	}
	if len(id) > 32 {
		id = id[:32]
	}
	return id
}
