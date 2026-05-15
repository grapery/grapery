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

// VerifyAppleReceiptRequest Apple 收据验证请求
type VerifyAppleReceiptRequest struct {
	ReceiptData string `json:"receipt_data" binding:"required"`
	Sandbox     bool   `json:"sandbox"`
}

// VerifyAppleReceiptResponse Apple 收据验证响应
type VerifyAppleReceiptResponse struct {
	Success bool            `json:"success"`
	Receipt *pay.IAPReceipt `json:"receipt,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// VerifyAppleReceipt 验证 Apple 收据
func (h *IAPHandler) VerifyAppleReceipt(c *gin.Context) {
	ctx := c.Request.Context()
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

	h.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"receipt_length": len(req.ReceiptData),
		"sandbox":        req.Sandbox,
		"endpoint":       "VerifyAppleReceipt",
	}).Info("Starting Apple receipt verification")

	// 验证收据
	receipt, err := h.iapService.VerifyReceipt(ctx, req.ReceiptData, req.Sandbox)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"user_id":        userID,
			"receipt_length": len(req.ReceiptData),
			"sandbox":        req.Sandbox,
			"endpoint":       "VerifyAppleReceipt",
			"error":          err.Error(),
		}).Error("Failed to verify Apple receipt")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "failed to verify receipt",
		})
		return
	}

	// 设置用户 ID
	receipt.UserID = userID

	h.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"receipt_id": receipt.ID,
		"bundle_id":  receipt.BundleID,
		"sandbox":    req.Sandbox,
		"endpoint":   "VerifyAppleReceipt",
	}).Info("Apple receipt verification completed successfully")

	// 充值 token：从产品表获取 QuotaLimit（单位：tokens），写入主库 memberships
	userIDStr := getUserIDString(c)
	if userIDStr != "" && receipt.ProductID != "" {
		if product, prodErr := h.productService.GetProductByProductID(ctx, receipt.ProductID); prodErr == nil && product != nil {
			tokenAmount := product.QuotaLimit
			if tokenAmount > 0 {
				if topUpErr := h.topUpUserTokens(ctx, userIDStr, tokenAmount); topUpErr != nil {
					h.logger.WithFields(logrus.Fields{
						"user_id":      userIDStr,
						"product_id":   receipt.ProductID,
						"token_amount": tokenAmount,
						"error":        topUpErr.Error(),
					}).Error("Failed to top up user tokens after IAP purchase")
				} else {
					h.logger.WithFields(logrus.Fields{
						"user_id":      userIDStr,
						"product_id":   receipt.ProductID,
						"token_amount": tokenAmount,
					}).Info("Successfully topped up user tokens after IAP purchase")
				}
			}
		} else if prodErr != nil {
			h.logger.WithFields(logrus.Fields{
				"user_id":    userIDStr,
				"product_id": receipt.ProductID,
				"error":      prodErr.Error(),
			}).Warn("Could not look up product for token top-up")
		}
	}

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
	Success      bool                 `json:"success"`
	Subscription *pay.IAPSubscription `json:"subscription,omitempty"`
	Error        string               `json:"error,omitempty"`
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
		var appleSub paymodels.AppleSubscription
		err = paymodels.DataBase().WithContext(ctx).
			Where("user_id = ?", userID).
			Order("updated_at DESC").
			First(&appleSub).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusOK, gin.H{
					"code": 0,
					"msg":  "success",
					"data": GetAppleSubscriptionStatusResponse{
						Success:        true,
						Subscription: nil,
					},
				})
				return
			}
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
		subscription = iapSubscriptionFromAppleRow(&appleSub)
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

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": GetAppleSubscriptionStatusResponse{
			Success:      true,
			Subscription: subscription,
		},
	})
}

// HandleAppleNotificationRequest 处理 Apple 通知请求
type HandleAppleNotificationRequest struct {
	SignedPayload string `json:"signed_payload" binding:"required"`
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

	h.logger.WithFields(logrus.Fields{
		"payload_length": len(req.SignedPayload),
		"endpoint":       "HandleAppleNotification",
		"ip":             c.ClientIP(),
	}).Info("Starting Apple notification processing")

	// 解析通知
	notificationData, err := pay.ParseAppleNotification(req.SignedPayload)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"payload_length": len(req.SignedPayload),
			"endpoint":       "HandleAppleNotification",
			"error":          err.Error(),
		}).Error("Failed to parse Apple notification")
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "failed to parse notification",
		})
		return
	}

	// 创建通知记录
	notification := &pay.IAPNotification{
		Platform:         pay.IAPPlatformApple,
		NotificationID:   notificationData.NotificationUUID,
		NotificationType: notificationData.NotificationType,
		Subtype:          notificationData.Subtype,
		Version:          "1.0",
		SignedPayload:    req.SignedPayload,
		RawData:          "", // TODO: 序列化通知数据
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
	ctx := c.Request.Context()
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
	ctx := c.Request.Context()
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

// topUpUserTokens 购买成功后在主库 memberships 表中充值 tokens。
// tokens > 0 表示充值（增加 token_quota）。
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
	return h.mainDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
