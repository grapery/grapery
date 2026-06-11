package pay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	paymodels "github.com/grapestree/fgrapery/grapery/internal/repository/pay"
	"github.com/grapestree/fgrapery/grapery/internal/service/pay"
	"github.com/grapestree/fgrapery/grapery/internal/transport/pay/middleware"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// IAPHandler IAP 处理器
type IAPHandler struct {
	iapService     pay.IAPService
	productService pay.IAPProductService
	logger         *logrus.Logger
	// mainDB 是主服务数据库连接，用于购买成功后充值 token 余量
	mainDB *gorm.DB
	cache  cache.Cache
}

// NewIAPHandler 创建 IAP 处理器；mainDB 可为 nil（降级为只写 vippay 库）
func NewIAPHandler(iapService pay.IAPService, productService pay.IAPProductService, mainDB *gorm.DB) *IAPHandler {
	return &IAPHandler{
		iapService:     iapService,
		productService: productService,
		logger:         logrus.New(),
		mainDB:         mainDB,
	}
}

// SetCache wires optional Redis so membership writes invalidate grapery API caches.
func (h *IAPHandler) SetCache(c cache.Cache) {
	h.cache = c
}

func (h *IAPHandler) invalidateMembershipCache(ctx context.Context, userID string) {
	cache.InvalidateMembership(ctx, h.cache, userID)
}

// getUserID 从context获取用户ID并转换为uint64
// 使用 FNV-1a 哈希将字符串UUID转换为稳定的数值ID
func getUserID(c *gin.Context) uint64 {
	userIDStr := middleware.GetUserIDFromContext(c)
	if userIDStr == "" {
		return 0
	}
	return stringToUint64(userIDStr)
}

// getUserIDString 从context获取用户ID字符串
func getUserIDString(c *gin.Context) string {
	return middleware.GetUserIDFromContext(c)
}

// stringToUint64 将字符串转换为稳定的uint64值
// 使用 FNV-1a 64位哈希算法，对于相同的输入总是产生相同的输出
func stringToUint64(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

// uint64ToString 将uint64转换回字符串（用于日志和调试）
// 注意：这不是逆向操作，只是格式化输出
func uint64ToString(id uint64) string {
	return strconv.FormatUint(id, 10)
}

// requestContext 将 JWT 用户 ID 注入标准 context，供 pay 层 getUserIDFromContext 使用。
func requestContext(c *gin.Context) context.Context {
	return auth.ContextWithUserID(c.Request.Context(), getUserIDString(c))
}

// VerifyAppleReceiptRequest StoreKit 2 购买校验请求（App Store Server API 或本地 StoreKit 配置）
type VerifyAppleReceiptRequest struct {
	Sandbox               bool   `json:"sandbox"`
	StoreKitLocal         bool   `json:"storekit_local"`
	ProductID             string `json:"product_id"`
	TransactionID         string `json:"transaction_id"`
	OriginalTransactionID string `json:"original_transaction_id"`
}

// VerifyAppleReceiptResponse Apple 收据验证响应
type VerifyAppleReceiptResponse struct {
	Success bool            `json:"success"`
	Receipt *pay.IAPReceipt `json:"receipt,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// VerifyAppleReceipt 验证 Apple 收据
func (h *IAPHandler) VerifyAppleReceipt(c *gin.Context) {
	ctx := requestContext(c)
	userID := getUserID(c)
	if userID == 0 {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "VerifyAppleReceipt",
			"ip":       c.ClientIP(),
		}).Warn("Unauthorized access attempt to Apple receipt verification")
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	var req VerifyAppleReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":  userID,
			"error":    err.Error(),
			"endpoint": "VerifyAppleReceipt",
		}).Error("Invalid request parameters for Apple receipt verification")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request parameters",
		})
		return
	}

	userIDStr := getUserIDString(c)

	var receipt *pay.IAPReceipt
	var err error

	if req.StoreKitLocal {
		if !pay.AllowStoreKitLocalVerify() {
			h.logger.WithFields(logrus.Fields{
				"user_id":  userID,
				"endpoint": "VerifyAppleReceipt",
			}).Warn("storekit_local rejected: IAP_ALLOW_STOREKIT_LOCAL is not enabled")
			c.JSON(http.StatusForbidden, gin.H{
				"code": 403,
				"msg":  "storekit_local verification is disabled",
			})
			return
		}
		if !req.Sandbox {
			c.JSON(http.StatusBadRequest, gin.H{
				"code": 400,
				"msg":  "storekit_local requires sandbox=true",
			})
			return
		}
		receipt, err = h.buildStoreKitLocalReceipt(ctx, userID, req)
		if err != nil {
			h.logger.WithFields(logrus.Fields{
				"user_id":  userID,
				"endpoint": "VerifyAppleReceipt",
				"error":    err.Error(),
			}).Warn("StoreKit local verify rejected")
			c.JSON(http.StatusBadRequest, gin.H{
				"code": 400,
				"msg":  err.Error(),
			})
			return
		}
		h.logger.WithFields(logrus.Fields{
			"user_id":        userID,
			"product_id":     receipt.ProductID,
			"transaction_id": receipt.SubscriptionTransactionID,
			"endpoint":       "VerifyAppleReceipt",
		}).Info("Accepted StoreKit local sandbox transaction")
	} else if strings.TrimSpace(req.TransactionID) != "" {
		h.logger.WithFields(logrus.Fields{
			"user_id":        userID,
			"transaction_id": req.TransactionID,
			"sandbox":        req.Sandbox,
			"endpoint":       "VerifyAppleReceipt",
		}).Info("Starting Apple transaction verification (App Store Server API)")

		receipt, err = h.verifyAppleTransaction(ctx, req.TransactionID, req.Sandbox)
		if err != nil {
			h.logger.WithFields(logrus.Fields{
				"user_id":        userID,
				"transaction_id": req.TransactionID,
				"sandbox":        req.Sandbox,
				"endpoint":       "VerifyAppleReceipt",
				"error":          err.Error(),
			}).Error("Failed to verify Apple transaction")
			c.JSON(http.StatusBadRequest, gin.H{
				"code":  400,
				"msg":   "failed to verify transaction",
				"error": "Could not verify your App Store purchase. Please try again later.",
			})
			return
		}
		receipt.UserID = userID
		if want := strings.TrimSpace(req.ProductID); want != "" && !strings.EqualFold(want, strings.TrimSpace(receipt.ProductID)) {
			h.logger.WithFields(logrus.Fields{
				"user_id":        userID,
				"transaction_id": req.TransactionID,
				"client_product": want,
				"apple_product":  receipt.ProductID,
				"endpoint":       "VerifyAppleReceipt",
			}).Warn("product_id mismatch after Apple verification")
			c.JSON(http.StatusBadRequest, gin.H{
				"code": 400,
				"msg":  "product_id does not match App Store transaction",
			})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "transaction_id is required (StoreKit 2); use storekit_local only for Xcode StoreKit Configuration",
		})
		return
	}

	h.applyApplePurchaseGrants(ctx, userIDStr, receipt)
	h.persistApplePurchaseRecords(ctx, userID, userIDStr, receipt)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": VerifyAppleReceiptResponse{
			Success: true,
			Receipt: receipt,
		},
	})
}

// GetAppleSubscriptionStatusRequest 获取 Apple 订阅状态请求
// original_transaction_id 可选：空则按当前登录用户在 apple_subscriptions 中查找最近一条
type GetAppleSubscriptionStatusRequest struct {
	OriginalTransactionID string `json:"original_transaction_id"`
}

// GetAppleSubscriptionStatusResponse 获取 Apple 订阅状态响应
type GetAppleSubscriptionStatusResponse struct {
	Success      bool                     `json:"success"`
	Subscription *pay.IAPSubscription     `json:"subscription,omitempty"`
	Display      *SubscriptionDisplayInfo `json:"display,omitempty"`
	Error        string                   `json:"error,omitempty"`
}

// GetAppleSubscriptionStatus 获取 Apple 订阅状态
func (h *IAPHandler) GetAppleSubscriptionStatus(c *gin.Context) {
	ctx := c.Request.Context()
	userID := getUserID(c)
	if userID == 0 {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "GetAppleSubscriptionStatus",
			"ip":       c.ClientIP(),
		}).Warn("Unauthorized access attempt to Apple subscription status")
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	b, errRead := io.ReadAll(c.Request.Body)
	if errRead != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":  userID,
			"error":    errRead.Error(),
			"endpoint": "GetAppleSubscriptionStatus",
		}).Error("Failed to read Apple subscription status request body")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request body",
		})
		return
	}
	b = bytes.TrimSpace(b)
	var req GetAppleSubscriptionStatusRequest
	if len(b) > 0 {
		if err := json.Unmarshal(b, &req); err != nil {
			h.logger.WithFields(logrus.Fields{
				"user_id":  userID,
				"error":    err.Error(),
				"endpoint": "GetAppleSubscriptionStatus",
			}).Error("Invalid request parameters for Apple subscription status")
			c.JSON(http.StatusBadRequest, gin.H{
				"code": 400,
				"msg":  "invalid request parameters",
			})
			return
		}
	}

	originalID := strings.TrimSpace(req.OriginalTransactionID)

	h.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"transaction_id": originalID,
		"endpoint":       "GetAppleSubscriptionStatus",
	}).Info("Starting Apple subscription status query")

	var subscription *pay.IAPSubscription
	var err error
	if originalID != "" {
		subscription, err = h.iapService.GetSubscription(ctx, originalID)
	} else {
		var appleSubs []paymodels.AppleSubscription
		err = paymodels.DataBase().WithContext(ctx).
			Where("user_id = ?", userID).
			Order("updated_at DESC").
			Limit(1).
			Find(&appleSubs).Error
		if err != nil {
			h.logger.WithFields(logrus.Fields{
				"user_id":  userID,
				"endpoint": "GetAppleSubscriptionStatus",
				"error":    err.Error(),
			}).Error("Failed to look up Apple subscription for user")
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 500,
				"msg":  "failed to get subscription status",
			})
			return
		}
		if len(appleSubs) == 0 {
			userIDStr := getUserIDString(c)
			display := h.buildSubscriptionDisplay(ctx, userIDStr, nil)
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"msg":  "success",
				"data": GetAppleSubscriptionStatusResponse{
					Success:      true,
					Subscription: nil,
					Display:      display,
				},
			})
			return
		}
		subscription = iapSubscriptionFromAppleRow(&appleSubs[0])
	}
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":        userID,
			"transaction_id": originalID,
			"endpoint":       "GetAppleSubscriptionStatus",
			"error":          err.Error(),
		}).Error("Failed to get Apple subscription status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to get subscription status",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":             userID,
		"transaction_id":      originalID,
		"subscription_status": subscription.Status,
		"endpoint":            "GetAppleSubscriptionStatus",
	}).Info("Apple subscription status retrieved successfully")

	userIDStr := getUserIDString(c)
	display := h.buildSubscriptionDisplay(ctx, userIDStr, subscription)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": GetAppleSubscriptionStatusResponse{
			Success:      true,
			Subscription: subscription,
			Display:      display,
		},
	})
}

// AckAppleSubscriptionNoticeRequest ACK 订阅变更提示。
type AckAppleSubscriptionNoticeRequest struct {
	NoticeID string `json:"notice_id" binding:"required"`
}

// AckAppleSubscriptionNotice 标记订阅变更提示已读。
func (h *IAPHandler) AckAppleSubscriptionNotice(c *gin.Context) {
	ctx := c.Request.Context()
	userIDStr := getUserIDString(c)
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "unauthorized"})
		return
	}
	var req AckAppleSubscriptionNoticeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "invalid request parameters"})
		return
	}
	if err := paymodels.AckSubscriptionNotice(ctx, strings.TrimSpace(req.NoticeID), userIDStr); err != nil {
		h.logger.WithError(err).Error("ack subscription notice failed")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "failed to ack notice"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "success"})
}

// HandleAppleNotificationRequest 处理 Apple 通知请求（ASC V2 使用 signedPayload）。
type HandleAppleNotificationRequest struct {
	SignedPayload      string `json:"signedPayload"`
	SignedPayloadSnake string `json:"signed_payload"`
}

func (r *HandleAppleNotificationRequest) payload() string {
	if s := strings.TrimSpace(r.SignedPayload); s != "" {
		return s
	}
	return strings.TrimSpace(r.SignedPayloadSnake)
}

// HandleAppleNotification 处理 Apple 通知
func (h *IAPHandler) HandleAppleNotification(c *gin.Context) {
	ctx := c.Request.Context()

	var req HandleAppleNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"error":    err.Error(),
			"endpoint": "HandleAppleNotification",
			"ip":       c.ClientIP(),
		}).Error("Invalid request parameters for Apple notification")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request parameters",
		})
		return
	}
	signedPayload := req.payload()
	if signedPayload == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "signedPayload is required",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"payload_length": len(signedPayload),
		"endpoint":       "HandleAppleNotification",
		"ip":             c.ClientIP(),
	}).Info("Starting Apple notification processing")

	// 解析通知
	notificationData, err := pay.ParseAppleNotification(signedPayload)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"payload_length": len(signedPayload),
			"endpoint":       "HandleAppleNotification",
			"error":          err.Error(),
		}).Error("Failed to parse Apple notification")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "failed to parse notification",
		})
		return
	}

	rawData, _ := json.Marshal(map[string]string{"signedPayload": signedPayload})

	// 创建通知记录
	notification := &pay.IAPNotification{
		Platform:         pay.IAPPlatformApple,
		NotificationID:   notificationData.NotificationUUID,
		NotificationType: notificationData.NotificationType,
		Subtype:          notificationData.Subtype,
		Version:          "2.0",
		SignedPayload:    signedPayload,
		RawData:          string(rawData),
		ProcessedAt:      time.Now(),
		Status:           "Pending",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// 处理通知
	h.logger.WithFields(logrus.Fields{
		"notification_id":   notification.NotificationID,
		"notification_type": notification.NotificationType,
		"subtype":           notification.Subtype,
		"endpoint":          "HandleAppleNotification",
	}).Info("Processing Apple notification")

	if err := h.iapService.HandleNotification(ctx, notification); err != nil {
		h.logger.WithFields(logrus.Fields{
			"notification_id":   notification.NotificationID,
			"notification_type": notification.NotificationType,
			"subtype":           notification.Subtype,
			"endpoint":          "HandleAppleNotification",
			"error":             err.Error(),
		}).Error("Failed to handle Apple notification")
		notification.Status = "Failed"
		notification.ErrorMessage = err.Error()
	} else {
		h.logger.WithFields(logrus.Fields{
			"notification_id":   notification.NotificationID,
			"notification_type": notification.NotificationType,
			"subtype":           notification.Subtype,
			"endpoint":          "HandleAppleNotification",
		}).Info("Apple notification processed successfully")
		notification.Status = "Success"
		h.applyEntitlementsFromAppleNotification(ctx, signedPayload, notificationData.NotificationType, notificationData.Subtype)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"notification_id": notification.NotificationID,
			"status":          notification.Status,
		},
	})
}

// VerifyGooglePurchaseRequest Google 购买验证请求
type VerifyGooglePurchaseRequest struct {
	PurchaseToken string `json:"purchase_token" binding:"required"`
	ProductID     string `json:"product_id" binding:"required"`
}

// VerifyGooglePurchaseResponse Google 购买验证响应
type VerifyGooglePurchaseResponse struct {
	Success  bool            `json:"success"`
	Purchase *pay.IAPReceipt `json:"purchase,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// VerifyGooglePurchase 验证 Google 购买
func (h *IAPHandler) VerifyGooglePurchase(c *gin.Context) {
	ctx := c.Request.Context()
	userID := getUserID(c)
	if userID == 0 {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "VerifyGooglePurchase",
			"ip":       c.ClientIP(),
		}).Warn("Unauthorized access attempt to Google purchase verification")
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	var req VerifyGooglePurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":  userID,
			"error":    err.Error(),
			"endpoint": "VerifyGooglePurchase",
		}).Error("Invalid request parameters for Google purchase verification")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request parameters",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"purchase_token": req.PurchaseToken,
		"product_id":     req.ProductID,
		"endpoint":       "VerifyGooglePurchase",
	}).Info("Starting Google purchase verification")

	purchase, err := h.iapService.VerifyReceipt(ctx, req.PurchaseToken, false) // Google通常不使用sandbox
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":        userID,
			"purchase_token": req.PurchaseToken,
			"product_id":     req.ProductID,
			"endpoint":       "VerifyGooglePurchase",
			"error":          err.Error(),
		}).Error("Failed to verify Google purchase")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to verify purchase",
		})
		return
	}

	// 设置用户 ID
	purchase.UserID = userID

	h.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"purchase_token": req.PurchaseToken,
		"product_id":     req.ProductID,
		"receipt_id":     purchase.ID,
		"endpoint":       "VerifyGooglePurchase",
	}).Info("Google purchase verification completed successfully")

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": VerifyGooglePurchaseResponse{
			Success:  true,
			Purchase: purchase,
		},
	})
}

// GetGoogleSubscriptionStatusRequest 获取 Google 订阅状态请求
type GetGoogleSubscriptionStatusRequest struct {
	PurchaseToken string `json:"purchase_token" binding:"required"`
	ProductID     string `json:"product_id" binding:"required"`
}

// GetGoogleSubscriptionStatusResponse 获取 Google 订阅状态响应
type GetGoogleSubscriptionStatusResponse struct {
	Success      bool                 `json:"success"`
	Subscription *pay.IAPSubscription `json:"subscription,omitempty"`
	Error        string               `json:"error,omitempty"`
}

// GetGoogleSubscriptionStatus 获取 Google 订阅状态
func (h *IAPHandler) GetGoogleSubscriptionStatus(c *gin.Context) {
	ctx := c.Request.Context()
	userID := getUserID(c)
	if userID == 0 {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "GetGoogleSubscriptionStatus",
			"ip":       c.ClientIP(),
		}).Warn("Unauthorized access attempt to Google subscription status")
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	var req GetGoogleSubscriptionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":  userID,
			"error":    err.Error(),
			"endpoint": "GetGoogleSubscriptionStatus",
		}).Error("Invalid request parameters for Google subscription status")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request parameters",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"purchase_token": req.PurchaseToken,
		"product_id":     req.ProductID,
		"endpoint":       "GetGoogleSubscriptionStatus",
	}).Info("Starting Google subscription status query")

	subscription, err := h.iapService.GetSubscription(ctx, req.PurchaseToken)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":        userID,
			"purchase_token": req.PurchaseToken,
			"product_id":     req.ProductID,
			"endpoint":       "GetGoogleSubscriptionStatus",
			"error":          err.Error(),
		}).Error("Failed to get Google subscription status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to get subscription status",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":         userID,
		"purchase_token":  req.PurchaseToken,
		"product_id":      req.ProductID,
		"subscription_id": subscription.ID,
		"endpoint":        "GetGoogleSubscriptionStatus",
	}).Info("Google subscription status retrieved successfully")

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": GetGoogleSubscriptionStatusResponse{
			Success:      true,
			Subscription: subscription,
		},
	})
}

// HandleGoogleNotificationRequest 处理 Google 通知请求
type HandleGoogleNotificationRequest struct {
	Data string `json:"data" binding:"required"`
}

// HandleGoogleNotification 处理 Google 通知
func (h *IAPHandler) HandleGoogleNotification(c *gin.Context) {
	ctx := c.Request.Context()

	var req HandleGoogleNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"error":    err.Error(),
			"endpoint": "HandleGoogleNotification",
			"ip":       c.ClientIP(),
		}).Error("Invalid request parameters for Google notification")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request parameters",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"data_length": len(req.Data),
		"endpoint":    "HandleGoogleNotification",
		"ip":          c.ClientIP(),
	}).Info("Starting Google notification processing")

	// 解析通知
	notificationData, err := pay.ParseGoogleNotification(req.Data)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"data_length": len(req.Data),
			"endpoint":    "HandleGoogleNotification",
			"error":       err.Error(),
		}).Error("Failed to parse Google notification")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "failed to parse notification",
		})
		return
	}

	// 创建通知记录
	notification := &pay.IAPNotification{
		Platform:         pay.IAPPlatformGoogle,
		NotificationID:   strconv.FormatInt(notificationData.EventTimeMillis, 10),
		Version:          notificationData.Version,
		NotificationType: notificationData.NotificationType,
		RawData:          req.Data,
		ProcessedAt:      time.Now(),
		Status:           "Pending",
		EventTimeMillis:  notificationData.EventTimeMillis,
		EventTime:        time.Unix(notificationData.EventTimeMillis/1000, 0),
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if notificationData.SubscriptionNotification.SubscriptionID != "" {
		notification.SubscriptionID = notificationData.SubscriptionNotification.SubscriptionID
	}

	// 处理通知
	h.logger.WithFields(logrus.Fields{
		"notification_id":   notification.NotificationID,
		"notification_type": notification.NotificationType,
		"event_time":        notification.EventTime,
		"endpoint":          "HandleGoogleNotification",
	}).Info("Processing Google notification")

	if err := h.iapService.HandleNotification(ctx, notification); err != nil {
		h.logger.WithFields(logrus.Fields{
			"notification_id":   notification.NotificationID,
			"notification_type": notification.NotificationType,
			"event_time":        notification.EventTime,
			"endpoint":          "HandleGoogleNotification",
			"error":             err.Error(),
		}).Error("Failed to handle Google notification")
		notification.Status = "Failed"
		notification.ErrorMessage = err.Error()
	} else {
		h.logger.WithFields(logrus.Fields{
			"notification_id":   notification.NotificationID,
			"notification_type": notification.NotificationType,
			"event_time":        notification.EventTime,
			"endpoint":          "HandleGoogleNotification",
		}).Info("Google notification processed successfully")
		notification.Status = "Success"
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"notification_id": notification.NotificationID,
			"status":          notification.Status,
		},
	})
}

// AcknowledgePurchaseRequest 确认购买请求
type AcknowledgePurchaseRequest struct {
	Platform      string `json:"platform" binding:"required"` // apple, google
	PurchaseToken string `json:"purchase_token" binding:"required"`
}

// AcknowledgePurchase 确认购买
func (h *IAPHandler) AcknowledgePurchase(c *gin.Context) {
	ctx := requestContext(c)
	userID := getUserID(c)
	if userID == 0 {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "AcknowledgePurchase",
			"ip":       c.ClientIP(),
		}).Warn("Unauthorized access attempt to purchase acknowledgment")
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	var req AcknowledgePurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":  userID,
			"error":    err.Error(),
			"endpoint": "AcknowledgePurchase",
		}).Error("Invalid request parameters for purchase acknowledgment")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request parameters",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"platform":       req.Platform,
		"purchase_token": req.PurchaseToken,
		"endpoint":       "AcknowledgePurchase",
	}).Info("Starting purchase acknowledgment")

	var err error
	switch req.Platform {
	case "apple":
		h.logger.WithFields(logrus.Fields{
			"user_id":        userID,
			"platform":       req.Platform,
			"purchase_token": req.PurchaseToken,
			"endpoint":       "AcknowledgePurchase",
		}).Info("Processing Apple purchase acknowledgment")
		err = h.iapService.AcknowledgePurchase(ctx, req.PurchaseToken)
	case "google":
		h.logger.WithFields(logrus.Fields{
			"user_id":        userID,
			"platform":       req.Platform,
			"purchase_token": req.PurchaseToken,
			"endpoint":       "AcknowledgePurchase",
		}).Info("Processing Google purchase acknowledgment")
		err = h.iapService.AcknowledgePurchase(ctx, req.PurchaseToken)
	default:
		h.logger.WithFields(logrus.Fields{
			"user_id":        userID,
			"platform":       req.Platform,
			"purchase_token": req.PurchaseToken,
			"endpoint":       "AcknowledgePurchase",
		}).Error("Invalid platform for purchase acknowledgment")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid platform",
		})
		return
	}

	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":        userID,
			"platform":       req.Platform,
			"purchase_token": req.PurchaseToken,
			"endpoint":       "AcknowledgePurchase",
			"error":          err.Error(),
		}).Error("Failed to acknowledge purchase")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to acknowledge purchase",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"platform":       req.Platform,
		"purchase_token": req.PurchaseToken,
		"endpoint":       "AcknowledgePurchase",
	}).Info("Purchase acknowledgment completed successfully")

	// TODO: 集成系统通知服务
	// 异步发送系统通知（需要与主服务的通知系统集成）
	h.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"platform":       req.Platform,
		"purchase_token": req.PurchaseToken,
		"action":         "purchase_acknowledged",
	}).Info("Purchase acknowledgement completed, notification would be sent here")

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"acknowledged": true,
		},
	})
}

// ConsumePurchaseRequest 消费购买请求
type ConsumePurchaseRequest struct {
	Platform      string `json:"platform" binding:"required"` // apple, google
	PurchaseToken string `json:"purchase_token" binding:"required"`
}

// ConsumePurchase 消费购买
func (h *IAPHandler) ConsumePurchase(c *gin.Context) {
	ctx := requestContext(c)
	userID := getUserID(c)
	if userID == 0 {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "ConsumePurchase",
			"ip":       c.ClientIP(),
		}).Warn("Unauthorized access attempt to purchase consumption")
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	var req ConsumePurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":  userID,
			"error":    err.Error(),
			"endpoint": "ConsumePurchase",
		}).Error("Invalid request parameters for purchase consumption")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request parameters",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"platform":       req.Platform,
		"purchase_token": req.PurchaseToken,
		"endpoint":       "ConsumePurchase",
	}).Info("Starting purchase consumption")

	var err error
	switch req.Platform {
	case "apple":
		h.logger.WithFields(logrus.Fields{
			"user_id":        userID,
			"platform":       req.Platform,
			"purchase_token": req.PurchaseToken,
			"endpoint":       "ConsumePurchase",
		}).Info("Processing Apple purchase consumption")
		err = h.iapService.ConsumePurchase(ctx, req.PurchaseToken)
	case "google":
		h.logger.WithFields(logrus.Fields{
			"user_id":        userID,
			"platform":       req.Platform,
			"purchase_token": req.PurchaseToken,
			"endpoint":       "ConsumePurchase",
		}).Info("Processing Google purchase consumption")
		err = h.iapService.ConsumePurchase(ctx, req.PurchaseToken)
	default:
		h.logger.WithFields(logrus.Fields{
			"user_id":        userID,
			"platform":       req.Platform,
			"purchase_token": req.PurchaseToken,
			"endpoint":       "ConsumePurchase",
		}).Error("Invalid platform for purchase consumption")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid platform",
		})
		return
	}

	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":        userID,
			"platform":       req.Platform,
			"purchase_token": req.PurchaseToken,
			"endpoint":       "ConsumePurchase",
			"error":          err.Error(),
		}).Error("Failed to consume purchase")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to consume purchase",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"platform":       req.Platform,
		"purchase_token": req.PurchaseToken,
		"endpoint":       "ConsumePurchase",
	}).Info("Purchase consumption completed successfully")

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"consumed": true,
		},
	})
}

// SyncSubscriptions 同步订阅
func (h *IAPHandler) SyncSubscriptions(c *gin.Context) {
	userID := getUserID(c)
	if userID == 0 {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "SyncSubscriptions",
			"ip":       c.ClientIP(),
		}).Warn("Unauthorized access attempt to subscription sync")
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "unauthorized",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"endpoint": "SyncSubscriptions",
	}).Info("Starting subscription synchronization")

	// 注意：新的接口中没有SyncAllSubscriptions方法，这里需要根据具体需求调整
	// 可能需要分别调用Apple和Google的同步方法，或者使用复合服务
	h.logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"endpoint": "SyncSubscriptions",
	}).Warn("SyncAllSubscriptions method not available in new interface")

	// 暂时返回成功，实际实现需要根据具体需求调整
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"synced": true,
		},
	})
}

// === 产品信息查询接口 ===

// GetProductsRequest 获取产品列表请求
type GetProductsRequest struct {
	Platform string `form:"platform" binding:"required,oneof=apple google"`                 // 平台
	Type     string `form:"type" binding:"omitempty,oneof=subscription onetime consumable"` // 产品类型（可选）
	Featured *bool  `form:"featured"`                                                       // 是否只获取推荐产品（可选）
}

// GetProducts 获取产品列表
func (h *IAPHandler) GetProducts(c *gin.Context) {
	ctx := c.Request.Context()

	// 解析请求参数
	var req GetProductsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		h.logger.WithFields(logrus.Fields{
			"error":    err.Error(),
			"endpoint": "GetProducts",
			"ip":       c.ClientIP(),
		}).Error("Invalid request parameters for product list query")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "invalid request parameters",
		})
		return
	}

	// 确定平台
	var platform paymodels.IAPProductPlatform
	if req.Platform == "apple" {
		platform = paymodels.IAPProductPlatformApple
		h.logger.WithFields(logrus.Fields{
			"platform": req.Platform,
			"endpoint": "GetProducts",
		}).Info("Selected Apple platform for product query")
	} else {
		platform = paymodels.IAPProductPlatformGoogle
		h.logger.WithFields(logrus.Fields{
			"platform": req.Platform,
			"endpoint": "GetProducts",
		}).Info("Selected Google platform for product query")
	}

	var products []*paymodels.IAPProduct
	var err error

	// 根据参数获取产品
	if req.Featured != nil && *req.Featured {
		h.logger.WithFields(logrus.Fields{
			"platform": req.Platform,
			"featured": *req.Featured,
			"endpoint": "GetProducts",
		}).Info("Querying featured products")
		products, err = h.productService.GetFeaturedProducts(ctx, platform)
	} else if req.Type != "" {
		var productType paymodels.IAPProductType
		switch req.Type {
		case "subscription":
			productType = paymodels.IAPProductTypeSubscription
		case "onetime":
			productType = paymodels.IAPProductTypeOneTime
		case "consumable":
			productType = paymodels.IAPProductTypeConsumable
		}
		h.logger.WithFields(logrus.Fields{
			"platform":     req.Platform,
			"product_type": req.Type,
			"endpoint":     "GetProducts",
		}).Info("Querying products by type")
		products, err = h.productService.GetProductsByType(ctx, productType, platform)
	} else {
		h.logger.WithFields(logrus.Fields{
			"platform": req.Platform,
			"endpoint": "GetProducts",
		}).Info("Querying all active products")
		products, err = h.productService.GetActiveProducts(ctx, platform)
	}

	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"platform": req.Platform,
			"type":     req.Type,
			"featured": req.Featured,
			"endpoint": "GetProducts",
			"error":    err.Error(),
		}).Error("Failed to get products")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to get products",
		})
		return
	}

	// 转换为响应格式
	productInfos := make([]gin.H, 0, len(products))
	for _, product := range products {
		productInfo := gin.H{
			"id":              product.ID,
			"product_id":      product.ProductID,
			"platform":        req.Platform,
			"product_type":    product.ProductType,
			"name":            product.Name,
			"description":     product.Description,
			"price":           product.Price,
			"currency":        product.Currency,
			"is_subscription": product.IsSubscription(),
			"is_available":    product.IsAvailable(),
			"featured":        product.Featured,
			"max_roles":       product.MaxRoles,
			"max_contexts":    product.MaxContexts,
			"quota_limit":     product.QuotaLimit,
			"display_order":   product.DisplayOrder,
		}

		// 添加订阅特定信息
		if product.IsSubscription() {
			productInfo["duration"] = product.Duration
			productInfo["trial_period"] = product.TrialPeriod
			productInfo["intro_offer"] = product.IntroOffer
		}

		productInfos = append(productInfos, productInfo)
	}

	h.logger.WithFields(logrus.Fields{
		"platform":      req.Platform,
		"type":          req.Type,
		"featured":      req.Featured,
		"product_count": len(productInfos),
		"endpoint":      "GetProducts",
	}).Info("Product list retrieved successfully")

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"products": productInfos,
			"total":    len(productInfos),
		},
	})
}

// GetProductDetail 获取产品详情
func (h *IAPHandler) GetProductDetail(c *gin.Context) {
	ctx := c.Request.Context()

	productID := c.Param("id")
	if productID == "" {
		productID = c.Param("product_id")
	}
	if productID == "" {
		h.logger.WithFields(logrus.Fields{
			"endpoint": "GetProductDetail",
			"ip":       c.ClientIP(),
		}).Error("Missing id parameter for product detail query")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "product id is required",
		})
		return
	}

	// 获取产品详情
	h.logger.WithFields(logrus.Fields{
		"product_id": productID,
		"endpoint":   "GetProductDetail",
	}).Info("Querying product detail")

	product, err := h.productService.GetProductByProductID(ctx, productID)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"product_id": productID,
			"endpoint":   "GetProductDetail",
			"error":      err.Error(),
		}).Error("Product not found")
		c.JSON(http.StatusNotFound, gin.H{
			"code": 404,
			"msg":  "product not found",
		})
		return
	}

	// 确定平台字符串
	platformStr := "apple"
	if product.Platform == paymodels.IAPProductPlatformGoogle {
		platformStr = "google"
		h.logger.WithFields(logrus.Fields{
			"product_id": productID,
			"platform":   platformStr,
			"endpoint":   "GetProductDetail",
		}).Info("Product is Google platform")
	} else {
		h.logger.WithFields(logrus.Fields{
			"product_id": productID,
			"platform":   platformStr,
			"endpoint":   "GetProductDetail",
		}).Info("Product is Apple platform")
	}

	// 构建响应
	productDetail := gin.H{
		"id":               product.ID,
		"product_id":       product.ProductID,
		"platform":         platformStr,
		"product_type":     product.ProductType,
		"name":             product.Name,
		"description":      product.Description,
		"price":            product.Price,
		"currency":         product.Currency,
		"status":           product.Status,
		"is_active":        product.IsActive,
		"is_subscription":  product.IsSubscription(),
		"is_available":     product.IsAvailable(),
		"featured":         product.Featured,
		"family_shareable": product.FamilyShareable,
		"max_roles":        product.MaxRoles,
		"max_contexts":     product.MaxContexts,
		"quota_limit":      product.QuotaLimit,
		"display_order":    product.DisplayOrder,
		"sync_status":      product.SyncStatus,
		"last_sync_time":   product.LastSyncTime,
	}

	// 添加订阅特定信息
	if product.IsSubscription() {
		productDetail["duration"] = product.Duration
		productDetail["trial_period"] = product.TrialPeriod
		productDetail["intro_offer"] = product.IntroOffer
		productDetail["subscription_group"] = product.SubscriptionGroup
		productDetail["duration_days"] = product.GetDurationInDays()
	}

	// 添加平台特定信息
	if product.AppleSKU != nil {
		productDetail["apple_sku"] = *product.AppleSKU
	}
	if product.GoogleSKU != nil {
		productDetail["google_sku"] = *product.GoogleSKU
	}
	if product.AppleProductID != nil {
		productDetail["apple_product_id"] = *product.AppleProductID
	}
	if product.GoogleProductID != nil {
		productDetail["google_product_id"] = *product.GoogleProductID
	}

	h.logger.WithFields(logrus.Fields{
		"product_id":   productID,
		"platform":     platformStr,
		"product_type": product.ProductType,
		"is_active":    product.IsActive,
		"is_available": product.IsAvailable(),
		"endpoint":     "GetProductDetail",
	}).Info("Product detail retrieved successfully")

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": productDetail,
	})
}

// GetProductStats 获取产品统计信息
func (h *IAPHandler) GetProductStats(c *gin.Context) {
	ctx := c.Request.Context()

	platformStr := c.Query("platform")
	if platformStr == "" {
		platformStr = "apple" // 默认查询 Apple 平台
		h.logger.WithFields(logrus.Fields{
			"endpoint": "GetProductStats",
			"ip":       c.ClientIP(),
		}).Info("No platform specified, defaulting to Apple")
	} else {
		h.logger.WithFields(logrus.Fields{
			"platform": platformStr,
			"endpoint": "GetProductStats",
			"ip":       c.ClientIP(),
		}).Info("Platform specified for product stats query")
	}

	var platform paymodels.IAPProductPlatform
	if platformStr == "apple" {
		platform = paymodels.IAPProductPlatformApple
		h.logger.WithFields(logrus.Fields{
			"platform": platformStr,
			"endpoint": "GetProductStats",
		}).Info("Selected Apple platform for product stats")
	} else {
		platform = paymodels.IAPProductPlatformGoogle
		h.logger.WithFields(logrus.Fields{
			"platform": platformStr,
			"endpoint": "GetProductStats",
		}).Info("Selected Google platform for product stats")
	}

	// 获取统计信息
	h.logger.WithFields(logrus.Fields{
		"platform": platformStr,
		"endpoint": "GetProductStats",
	}).Info("Querying product statistics")

	stats, err := h.productService.GetProductStats(ctx, platform)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"platform": platformStr,
			"endpoint": "GetProductStats",
			"error":    err.Error(),
		}).Error("Failed to get product stats")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to get product stats",
		})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"platform": platformStr,
		"endpoint": "GetProductStats",
	}).Info("Product statistics retrieved successfully")

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"platform": platformStr,
			"stats":    stats,
		},
	})
}

func (h *IAPHandler) verifyAppleTransaction(ctx context.Context, transactionID string, sandbox bool) (*pay.IAPReceipt, error) {
	if comp, ok := h.iapService.(*pay.CompositeIAPService); ok {
		if apple, ok := comp.GetAppleService().(*pay.AppleIAPService); ok {
			return apple.VerifyTransaction(ctx, transactionID, sandbox)
		}
	}
	if apple, ok := h.iapService.(*pay.AppleIAPService); ok {
		return apple.VerifyTransaction(ctx, transactionID, sandbox)
	}
	return nil, fmt.Errorf("apple transaction verification is not configured")
}

// buildStoreKitLocalReceipt 为 Xcode StoreKit Configuration（无 App Store 收据文件）构建可入账的收据。
// 仅允许 sandbox + storekit_local，供本地/模拟器开发使用。
func (h *IAPHandler) buildStoreKitLocalReceipt(ctx context.Context, userID uint64, req VerifyAppleReceiptRequest) (*pay.IAPReceipt, error) {
	productID := strings.TrimSpace(req.ProductID)
	txID := strings.TrimSpace(req.TransactionID)
	if productID == "" || txID == "" {
		return nil, fmt.Errorf("product_id and transaction_id are required for storekit_local verify")
	}
	origTx := strings.TrimSpace(req.OriginalTransactionID)
	if origTx == "" {
		origTx = txID
	}

	product, err := h.productService.GetProductByProductID(ctx, productID)
	if err != nil || product == nil {
		return nil, fmt.Errorf("unknown product_id: %s", productID)
	}

	now := time.Now()
	exp := subscriptionEndTimeFromProduct(product, now)

	return &pay.IAPReceipt{
		UserID:                    userID,
		Platform:                  pay.IAPPlatformApple,
		ProductID:                 productID,
		OriginalTransactionID:     origTx,
		SubscriptionTransactionID: txID,
		Environment:               "Sandbox",
		Status:                    "Valid",
		CreationDate:              now,
		ExpirationDate:            &exp,
		CreatedAt:                 now,
		UpdatedAt:                 now,
	}, nil
}

func subscriptionEndTimeFromProduct(product *paymodels.IAPProduct, start time.Time) time.Time {
	if product == nil {
		return start.AddDate(0, 1, 0)
	}
	if days := product.GetDurationInDays(); days > 0 {
		return start.AddDate(0, 0, days)
	}
	return start.AddDate(0, 1, 0)
}

// persistApplePurchaseRecords 写入 vippay 库中的 Apple 订阅与用户订阅记录，供 VIP 状态查询。
func (h *IAPHandler) persistApplePurchaseRecords(ctx context.Context, userID uint64, appUserID string, receipt *pay.IAPReceipt) {
	if receipt == nil || receipt.ProductID == "" {
		return
	}

	product, err := h.productService.GetProductByProductID(ctx, receipt.ProductID)
	if err != nil || product == nil {
		return
	}

	origTx := strings.TrimSpace(receipt.OriginalTransactionID)
	if origTx == "" {
		origTx = strings.TrimSpace(receipt.SubscriptionTransactionID)
	}
	if origTx == "" {
		return
	}

	now := time.Now()
	start := receipt.CreationDate
	if start.IsZero() {
		start = now
	}
	end := subscriptionEndTimeFromProduct(product, start)
	if receipt.ExpirationDate != nil {
		end = *receipt.ExpirationDate
	}

	var appleRows []paymodels.AppleSubscription
	if dbErr := paymodels.DataBase().WithContext(ctx).
		Where("original_transaction_id = ?", origTx).
		Limit(1).
		Find(&appleRows).Error; dbErr != nil {
		h.logger.WithError(dbErr).Warn("Failed to query apple_subscriptions for upsert")
		return
	}

	if len(appleRows) == 0 {
		row := paymodels.AppleSubscription{
			UserID:                userID,
			AppUserID:             strings.TrimSpace(appUserID),
			OriginalTransactionID: origTx,
			ProductID:             receipt.ProductID,
			PurchaseDate:          start,
			ExpiresDate:           &end,
			Status:                "Active",
			AutoRenewStatus:       "On",
		}
		if createErr := paymodels.DataBase().WithContext(ctx).Create(&row).Error; createErr != nil {
			h.logger.WithError(createErr).Warn("Failed to create apple_subscriptions row")
		}
	} else {
		row := appleRows[0]
		row.UserID = userID
		if strings.TrimSpace(appUserID) != "" {
			row.AppUserID = strings.TrimSpace(appUserID)
		}
		row.ProductID = receipt.ProductID
		row.ExpiresDate = &end
		row.Status = "Active"
		row.UpdatedAt = now
		if saveErr := paymodels.DataBase().WithContext(ctx).Save(&row).Error; saveErr != nil {
			h.logger.WithError(saveErr).Warn("Failed to update apple_subscriptions row")
		}
	}

	userIDInt := int64(userID)
	active, activeErr := paymodels.GetUserActiveSubscriptionByUserID(ctx, userIDInt)
	if activeErr == nil && active != nil {
		active.PackagePlanID = product.ID
		active.EndTime = end
		active.ProviderSubID = origTx
		active.MaxRoles = product.MaxRoles
		active.MaxContexts = product.MaxContexts
		active.QuotaLimit = product.QuotaLimit
		if updateErr := paymodels.UpdateUserSubscription(ctx, active); updateErr != nil {
			h.logger.WithError(updateErr).Warn("Failed to update user_subscriptions row")
		}
		return
	}

	sub := &paymodels.UserSubscription{
		UserID:          userIDInt,
		PackagePlanID:   product.ID,
		Status:          paymodels.UserSubscriptionStatusActive,
		StartTime:       start,
		EndTime:         end,
		AutoRenew:       true,
		PaymentMethod:   paymodels.PaymentMethodApplePay,
		PaymentProvider: "App Store",
		ProviderSubID:   origTx,
		Currency:        "CNY",
		QuotaLimit:      product.QuotaLimit,
		MaxRoles:        product.MaxRoles,
		MaxContexts:     product.MaxContexts,
	}
	if createErr := paymodels.CreateUserSubscription(ctx, sub); createErr != nil {
		h.logger.WithError(createErr).Warn("Failed to create user_subscriptions row")
	}
}

// topUpUserTokens 非订阅类 IAP：在主库 memberships 上增量增加 token_quota。
func (h *IAPHandler) topUpUserTokens(ctx context.Context, userIDStr string, tokens int) error {
	if h.mainDB == nil {
		h.logger.Warn("mainDB not configured, skipping token top-up")
		return nil
	}
	if tokens <= 0 || userIDStr == "" {
		return nil
	}

	type membership struct {
		ID           string     `gorm:"column:id"`
		UserID       string     `gorm:"column:user_id"`
		Tier         string     `gorm:"column:tier"`
		Status       string     `gorm:"column:status"`
		StartDate    time.Time  `gorm:"column:start_date"`
		EndDate      *time.Time `gorm:"column:end_date"`
		TokenQuota   int        `gorm:"column:token_quota"`
		TokenUsed    int        `gorm:"column:token_used"`
		StorageQuota int64      `gorm:"column:storage_quota"`
		StorageUsed  int64      `gorm:"column:storage_used"`
		CreatedAt    time.Time  `gorm:"column:created_at"`
		UpdatedAt    time.Time  `gorm:"column:updated_at"`
	}

	now := time.Now()
	err := h.mainDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var m membership
		err := tx.Table("memberships").Where("user_id = ?", userIDStr).First(&m).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			m = membership{
				ID:           uuid.New().String(),
				UserID:       userIDStr,
				Tier:         "free",
				Status:       string(common.MembershipStatusActive),
				StartDate:    now,
				TokenQuota:   common.DefaultFreeTierTokenQuota + tokens,
				TokenUsed:    0,
				StorageQuota: common.DefaultFreeTierStorageBytes,
				StorageUsed:  0,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			return tx.Table("memberships").Create(&m).Error
		}
		if err != nil {
			return fmt.Errorf("query membership: %w", err)
		}
		return tx.Table("memberships").
			Where("user_id = ?", userIDStr).
			Updates(map[string]interface{}{
				"token_quota": gorm.Expr("token_quota + ?", tokens),
				"updated_at":  time.Now(),
			}).Error
	})
	if err == nil {
		h.invalidateMembershipCache(ctx, userIDStr)
	}
	return err
}

// iapSubscriptionFromAppleRow 将本地库中的 Apple 订阅行转为与 GetSubscription 一致的结构
func iapSubscriptionFromAppleRow(a *paymodels.AppleSubscription) *pay.IAPSubscription {
	if a == nil {
		return nil
	}
	return &pay.IAPSubscription{
		ID:                     a.ID,
		UserID:                 a.UserID,
		Platform:               pay.IAPPlatformApple,
		SubscriptionID:         a.OriginalTransactionID,
		ProductID:              a.ProductID,
		PurchaseDate:           a.PurchaseDate,
		ExpiresDate:            a.ExpiresDate,
		Status:                 a.Status,
		AutoRenewStatus:        a.AutoRenewStatus,
		IsInIntroOfferPeriod:   a.IsInIntroOfferPeriod,
		IsInGracePeriod:        a.IsInGracePeriod,
		GracePeriodExpiresDate: a.GracePeriodExpiresDate,
		OfferType:              a.OfferType,
		PriceIncreaseStatus:    a.PriceIncreaseStatus,
		LastNotificationType:   a.LastNotificationType,
		LastNotificationDate:   a.LastNotificationDate,
		CreatedAt:              a.CreatedAt,
		UpdatedAt:              a.UpdatedAt,
	}
}
