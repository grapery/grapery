package auth

import (
	// "net/http"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	connect "connectrpc.com/connect"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/auth"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/pkg/auth"
	utils "github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/errors"
	"github.com/grapery/grapery/utils/jwt"
	"github.com/grapery/grapery/utils/log"
)

func AuthInterceptor(authFunc grpc_auth.AuthFunc) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		logger := log.Log().With(zap.String("method", info.FullMethod))
		logger.Info("AuthInterceptor called", zap.String("fullMethod", info.FullMethod))

		// 检查是否为免认证的方法
		if info.FullMethod == "/common.TeamsAPI/Login" ||
			info.FullMethod == "/common.TeamsAPI/About" ||
			info.FullMethod == "/common.TeamsAPI/Register" ||
			info.FullMethod == "/common.TeamsAPI/Reset_password" {
			logger.Info("Skipping authentication for public method")
			return handler(ctx, req)
		}

		var newCtx context.Context
		var err error
		if overrideSrv, ok := info.Server.(grpc_auth.ServiceAuthFuncOverride); ok {
			logger.Debug("Using service auth override")
			newCtx, err = overrideSrv.AuthFuncOverride(ctx, info.FullMethod)
		} else {
			logger.Debug("Using default auth function")
			newCtx, err = authFunc(ctx)
		}
		if err != nil {
			logger.Error("Authentication failed", zap.Error(err))
			return nil, err
		}

		logger.Info("Authentication successful, proceeding with handler")
		return handler(newCtx, req)
	}
}

type AuthInterceptorFunc struct {
	Handle func(context.Context, connect.Spec, http.Header, any) (context.Context, error)
}

func (f AuthInterceptorFunc) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		logger := log.Log().With(zap.String("procedure", req.Spec().Procedure))
		logger.Info("AuthInterceptorFunc.WrapUnary called", zap.String("procedure", req.Spec().Procedure))

		// 检查是否为免认证的方法
		if req.Spec().Procedure == "/rankquantity.voyager.api.TeamsAPI/Login" ||
			req.Spec().Procedure == "/rankquantity.voyager.api.TeamsAPI/About" ||
			req.Spec().Procedure == "/rankquantity.voyager.api.TeamsAPI/Register" ||
			req.Spec().Procedure == "/rankquantity.voyager.api.TeamsAPI/Reset_password" {
			logger.Info("Skipping authentication for public procedure")
			return next(ctx, req)
		}
		logger.Debug("Processing authentication for protected procedure")
		newCtx, err := f.Handle(ctx, req.Spec(), req.Header(), req)
		if err != nil {
			logger.Error("Authentication failed in WrapUnary", zap.Error(err))
			return nil, err
		}

		logger.Info("Authentication successful in WrapUnary, proceeding with handler")
		return next(newCtx, req)
	}
}

func (f AuthInterceptorFunc) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}
func (f AuthInterceptorFunc) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

func ConnectAuthFunc(ctx context.Context, spec connect.Spec, header http.Header, a any) (context.Context, error) {
	logger := log.Log().With(zap.String("procedure", spec.Procedure))
	logger.Info("ConnectAuthFunc called", zap.String("procedure", spec.Procedure))

	cookieInfo := header.Get(utils.GrpcGateWayCookie)
	if len(cookieInfo) == 0 {
		logger.Error("Empty auth cookie", zap.String("cookieName", utils.GrpcGateWayCookie))
		return nil, status.Errorf(codes.Unauthenticated, "empty auth from md: %s", utils.GrpcGateWayCookie)
	}

	logger.Debug("Found auth cookie", zap.String("cookieValue", cookieInfo))
	token := cookieInfo
	jwtInfo := jwt.NewJwtWrapper(utils.SecretKey, utils.ExpirationHours)
	tokenInfo, err := jwtInfo.ValidateToken(token)
	if err != nil {
		logger.Error("Invalid auth token", zap.Error(err), zap.String("token", token))
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}

	logger.Info("Token validation successful", zap.Int64("userId", tokenInfo.UID))
	header.Set("auth.sub", jwtInfo.SecretKey)
	header.Set(utils.UserIdKey, fmt.Sprintf("%d", tokenInfo.UID))

	// 将 user_id 存入 context
	newCtx := context.WithValue(ctx, utils.UserIdKey, tokenInfo.UID)

	// 记录请求参数
	aData, _ := json.Marshal(a)
	logger.Debug("Request parameters", zap.String("params", string(aData)))

	logger.Info("ConnectAuthFunc completed successfully")
	return newCtx, nil
}

func WithRequestLogInterceptor(ctx context.Context, spec connect.Spec, header http.Header, a any) error {
	logger := log.Log().With(zap.String("procedure", spec.Procedure))
	logger.Info("WithRequestLogInterceptor called", zap.String("procedure", spec.Procedure))

	aData := a.([]byte)
	logger.Debug("Request log interceptor", zap.String("params", string(aData)))

	logger.Info("WithRequestLogInterceptor completed")
	return nil
}

func AuthFunc(ctx context.Context) (context.Context, error) {
	logger := log.Log()
	logger.Info("AuthFunc called")

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		logger.Error("Failed to get metadata from incoming context")
		return nil, status.Errorf(codes.Unauthenticated, "metadata.FromIncomingContext: %v", codes.FailedPrecondition)
	}

	tokenList := md[utils.GrpcGateWayCookie]
	if len(tokenList) <= 0 {
		logger.Error("Empty auth token list", zap.String("cookieName", utils.GrpcGateWayCookie))
		return nil, fmt.Errorf("empty auth from md: %s", utils.GrpcGateWayCookie)
	}

	logger.Debug("Found token list", zap.Strings("tokens", tokenList))
	tokenListTemp := strings.Split(tokenList[0], "=")
	if len(tokenListTemp) < 2 {
		logger.Error("Invalid token format", zap.String("tokenString", tokenList[0]))
		return nil, fmt.Errorf("invalid token format")
	}

	token := tokenListTemp[1]
	logger.Debug("Extracted token", zap.String("token", token))

	jwtInfo := jwt.NewJwtWrapper(utils.SecretKey, utils.ExpirationHours)
	tokenInfo, err := jwtInfo.ValidateToken(token)
	if err != nil {
		logger.Error("Token validation failed", zap.Error(err), zap.String("token", token))
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", err)
	}

	logger.Info("Token validation successful", zap.Int64("userId", tokenInfo.UID))
	grpc_ctxtags.Extract(ctx).Set("auth.sub", jwtInfo.SecretKey)
	newCtx := context.WithValue(ctx, utils.UserIdKey, tokenInfo.UID)

	logger.Info("AuthFunc completed successfully")
	return newCtx, nil
}

type Result struct {
	Code  int    `json:"code,omitempty"`
	Token string `json:"token,omitempty"`
	Error string `json:"error,omitempty"`
}

func LoginFunc(w http.ResponseWriter, r *http.Request) {
	logger := log.Log().With(zap.String("function", "LoginFunc"))
	logger.Info("LoginFunc called", zap.String("method", r.Method), zap.String("url", r.URL.String()))

	auth := NewAuthService(utils.SecretKey, utils.ExpirationHours)
	reqBody, err := io.ReadAll(r.Body)
	ret := new(Result)
	if err != nil {
		logger.Error("Failed to read request body", zap.Error(err))
		ret.Code = int(api.ResponseCode_INVALID_PARAMETER)
		ret.Error = "参数错误"
		resultData, _ := json.Marshal(ret)
		_, _ = w.Write(resultData)
		return
	}

	logger.Debug("Request body read successfully", zap.Int("bodySize", len(reqBody)))
	info := &api.LoginRequest{}
	err = json.Unmarshal(reqBody, info)
	if err != nil {
		logger.Error("Failed to unmarshal request body", zap.Error(err), zap.String("body", string(reqBody)))
		ret.Code = int(api.ResponseCode_INVALID_PARAMETER)
		ret.Error = "参数错误"
		resultData, _ := json.Marshal(ret)
		_, _ = w.Write(resultData)
		return
	}

	logger.Debug("Request unmarshaled successfully", zap.String("account", info.Account))
	if info.Account == "" || info.Password == "" {
		logger.Error("Empty account or password", zap.String("account", info.Account), zap.Bool("hasPassword", info.Password != ""))
		ret.Code = int(api.ResponseCode_INVALID_PARAMETER)
		ret.Error = "账号或密码参数错误"
		resultData, _ := json.Marshal(ret)
		_, _ = w.Write(resultData)
		return
	}

	req := &connect.Request[api.LoginRequest]{
		Msg: info,
	}
	logger.Info("Calling auth.Login", zap.String("account", info.Account))
	resp, err := auth.Login(r.Context(), req)
	if err != nil {
		logger.Error("Login failed", zap.Error(err), zap.String("account", info.Account))
		ret.Code = int(api.ResponseCode_OPERATION_FAILED)
		ret.Error = err.Error()
		resultData, _ := json.Marshal(ret)
		w.Write(resultData)
		return
	}

	// 根据响应码设置不同的返回信息
	switch resp.Msg.GetCode() {
	case api.ResponseCode_OK:
		ret.Code = int(api.ResponseCode_OK)
		ret.Token = resp.Msg.GetData().GetToken()
		ret.Error = ""
	case api.ResponseCode_ACCOUNT_NOT_FOUND:
		ret.Code = int(api.ResponseCode_ACCOUNT_NOT_FOUND)
		ret.Error = "账号不存在"
	case api.ResponseCode_WRONG_PASSWORD:
		ret.Code = int(api.ResponseCode_WRONG_PASSWORD)
		ret.Error = "密码错误"
	case api.ResponseCode_OPERATION_FAILED:
		ret.Code = int(api.ResponseCode_OPERATION_FAILED)
		ret.Error = "登录失败"
	case api.ResponseCode_TOKEN_EXPIRED:
		ret.Code = int(api.ResponseCode_TOKEN_EXPIRED)
		ret.Error = "令牌已过期"
	case api.ResponseCode_TOKEN_INVALID:
		ret.Code = int(api.ResponseCode_TOKEN_INVALID)
		ret.Error = "令牌无效"
	case api.ResponseCode_RATE_LIMIT_EXCEEDED:
		ret.Code = int(api.ResponseCode_RATE_LIMIT_EXCEEDED)
		ret.Error = "请求频率过高"
	case api.ResponseCode_SERVICE_UNAVAILABLE:
		ret.Code = int(api.ResponseCode_SERVICE_UNAVAILABLE)
		ret.Error = "服务不可用"
	case api.ResponseCode_INTERNAL_ERROR:
		ret.Code = int(api.ResponseCode_INTERNAL_ERROR)
		ret.Error = "服务器内部错误"
	default:
		ret.Code = int(api.ResponseCode_OPERATION_FAILED)
		ret.Error = "登录失败"
	}

	resultData, _ := json.Marshal(ret)
	w.Header().Add("Content-Type", "application/json")
	if ret.Token != "" {
		w.Header().Add("Cookie", "token="+ret.Token)
	}
	_, _ = w.Write(resultData)

	logger.Info("LoginFunc completed successfully")
	return
}

func ParseInt(s string) int {
	logger := log.Log().With(zap.String("function", "ParseInt"))
	logger.Debug("ParseInt called", zap.String("input", s))

	if len(s) == 0 {
		logger.Debug("Empty string, returning 0")
		return 0
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		logger.Error("Failed to parse string to int", zap.Error(err), zap.String("input", s))
		return 0
	}

	logger.Debug("ParseInt completed", zap.Int("result", val))
	return val
}

type AuthService struct {
	Jwt *jwt.JwtWrapper
}

func NewAuthService(key string, expiration int) *AuthService {
	logger := log.Log().With(zap.String("function", "NewAuthService"))
	logger.Info("Creating new AuthService", zap.String("key", key), zap.Int("expiration", expiration))

	authService := &AuthService{
		Jwt: jwt.NewJwtWrapper(key, expiration),
	}

	logger.Info("AuthService created successfully")
	return authService
}

func (ts *AuthService) Login(ctx context.Context, req *connect.Request[api.LoginRequest]) (*connect.Response[api.LoginResponse], error) {
	logger := log.Log().With(zap.String("function", "AuthService.Login"))
	logger.Info("AuthService.Login called", zap.String("account", req.Msg.GetAccount()))

	info, err := auth.GetAuthService().
		Login(ctx, req.Msg.GetAccount(), req.Msg.GetPassword())
	if err != nil {
		logger.Error("Auth service login failed", zap.Error(err), zap.String("account", req.Msg.GetAccount()))

		// 使用响应码映射函数获取对应的响应码和消息
		responseCode := errors.GetAuthResponseCode(err)
		responseMsg := errors.GetAuthResponseMessage(responseCode)

		return &connect.Response[api.LoginResponse]{
			Msg: &api.LoginResponse{
				Code: responseCode,
				Msg:  responseMsg,
				Data: nil,
			},
		}, nil
	}

	logger.Info("Auth service login successful", zap.Int64("userId", info.GetUserId()), zap.String("account", req.Msg.GetAccount()))
	token, err := ts.Jwt.GenerateToken(info)
	if err != nil {
		logger.Error("Failed to generate JWT token", zap.Error(err), zap.Int64("userId", info.GetUserId()))
		return &connect.Response[api.LoginResponse]{
			Msg: &api.LoginResponse{
				Code: api.ResponseCode_OPERATION_FAILED,
				Msg:  "生成令牌失败",
				Data: nil,
			},
		}, nil
	}

	logger.Info("JWT token generated successfully", zap.Int64("userId", info.GetUserId()))
	ret := &api.LoginResponse{
		Code: api.ResponseCode_OK,
		Msg:  "登录成功",
		Data: &api.LoginResponse_Data{
			UserId: info.GetUserId(),
			Token:  token,
		},
	}

	logger.Info("AuthService.Login completed successfully", zap.Int64("userId", info.GetUserId()))
	return &connect.Response[api.LoginResponse]{Msg: ret}, nil
}

func (ts *AuthService) Logout(ctx context.Context, req *connect.Request[api.LogoutRequest]) (*connect.Response[api.LogoutResponse], error) {
	logger := log.Log().With(zap.String("function", "AuthService.Logout"))
	logger.Info("AuthService.Logout called")

	_, err := auth.GetAuthService().Logout(ctx, req.Msg)
	if err != nil {
		logger.Error("Auth service logout failed", zap.Error(err))
		return &connect.Response[api.LogoutResponse]{
			Msg: &api.LogoutResponse{
				Code: api.ResponseCode_OPERATION_FAILED,
				Msg:  "登出失败",
			},
		}, nil
	}

	logger.Info("AuthService.Logout completed successfully")
	return &connect.Response[api.LogoutResponse]{
		Msg: &api.LogoutResponse{
			Code: api.ResponseCode_OK,
			Msg:  "登出成功",
		},
	}, nil
}

func (ts *AuthService) Register(ctx context.Context, req *connect.Request[api.RegisterRequest]) (*connect.Response[api.RegisterResponse], error) {
	logger := log.Log().With(zap.String("function", "AuthService.Register"))
	logger.Info("AuthService.Register called", zap.String("name", req.Msg.GetName()), zap.String("account", req.Msg.GetAccount()))

	err := auth.GetAuthService().Register(
		context.Background(),
		req.Msg.GetName(),
		req.Msg.GetAccount(),
		req.Msg.GetPassword(),
	)
	if err != nil {
		logger.Error("Auth service register failed", zap.Error(err), zap.String("account", req.Msg.GetAccount()))

		// 使用响应码映射函数获取对应的响应码和消息
		responseCode := errors.GetAuthResponseCode(err)
		responseMsg := errors.GetAuthResponseMessage(responseCode)

		return &connect.Response[api.RegisterResponse]{
			Msg: &api.RegisterResponse{
				Code: responseCode,
				Msg:  responseMsg,
			},
		}, nil
	}

	logger.Info("AuthService.Register completed successfully", zap.String("account", req.Msg.GetAccount()))
	return &connect.Response[api.RegisterResponse]{
		Msg: &api.RegisterResponse{
			Code: api.ResponseCode_OK,
			Msg:  "注册成功",
		},
	}, nil
}

func (ts *AuthService) ResetPwd(ctx context.Context, req *connect.Request[api.ResetPasswordRequest]) (*connect.Response[api.ResetPasswordResponse], error) {
	logger := log.Log().With(zap.String("function", "AuthService.ResetPwd"))
	logger.Info("AuthService.ResetPwd called")

	resp, err := auth.GetAuthService().ResetPassword(ctx, req.Msg)
	if err != nil {
		logger.Error("Auth service reset password failed", zap.Error(err))
		return &connect.Response[api.ResetPasswordResponse]{
			Msg: &api.ResetPasswordResponse{
				Account:   req.Msg.GetAccount(),
				Status:    int64(api.ResponseCode_OPERATION_FAILED),
				Timestamp: time.Now().Unix(),
			},
		}, nil
	}

	logger.Info("AuthService.ResetPwd completed successfully")
	return &connect.Response[api.ResetPasswordResponse]{
		Msg: resp,
	}, nil
}

func (ts *AuthService) RefreshToken(ctx context.Context, req *connect.Request[api.RefreshTokenRequest]) (*connect.Response[api.RefreshTokenResponse], error) {
	logger := log.Log().With(zap.String("function", "AuthService.RefreshToken"))
	logger.Info("AuthService.RefreshToken called")

	token, err := ts.Jwt.ValidateToken(req.Msg.GetToken())
	if err != nil {
		logger.Error("Failed to validate refresh token", zap.Error(err))
		return nil, err
	}

	logger.Debug("RefreshToken request", zap.String("request", req.Msg.String()))
	info, err := auth.GetAuthService().GetUserInfo(ctx, token.UID, token.Email)
	if err != nil {
		logger.Error("Failed to get user info for refresh token", zap.Error(err), zap.Int64("userId", token.UID))
		return nil, err
	}

	infoData, _ := json.Marshal(info)
	logger.Debug("User info for refresh token", zap.String("userInfo", string(infoData)))

	if info == nil {
		logger.Error("User not found for refresh token", zap.Int64("userId", token.UID))
		return nil, fmt.Errorf("user not found")
	}

	if info.GetUserId() != token.UID {
		logger.Error("User ID mismatch in refresh token", zap.Int64("tokenUID", token.UID), zap.Int64("infoUID", info.GetUserId()))
		return nil, fmt.Errorf("user id not match")
	}

	newToken, err := ts.Jwt.GenerateToken(info)
	if err != nil {
		logger.Error("Failed to generate new token", zap.Error(err), zap.Int64("userId", info.GetUserId()))
		return nil, err
	}

	logger.Info("New token generated successfully", zap.Int64("userId", info.GetUserId()))
	ret := &api.RefreshTokenResponse{
		UserId: info.GetUserId(),
		Token:  newToken,
	}

	logger.Info("AuthService.RefreshToken completed successfully", zap.Int64("userId", info.GetUserId()))
	return &connect.Response[api.RefreshTokenResponse]{Msg: ret}, nil
}

// GetUserIDFromContext 从 context 中获取 user_id
func GetUserIDFromContext(ctx context.Context) (int64, error) {
	logger := log.Log().With(zap.String("function", "GetUserIDFromContext"))
	logger.Debug("GetUserIDFromContext called")

	userID, ok := ctx.Value(utils.UserIdKey).(int64)
	if !ok {
		logger.Error("User ID not found in context", zap.String("key", utils.UserIdKey))
		return 0, fmt.Errorf("user id not found in context")
	}

	logger.Debug("User ID retrieved from context", zap.Int64("userId", userID))
	return userID, nil
}
