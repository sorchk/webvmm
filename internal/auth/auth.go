package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/webvmm/webvmm/internal/models"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrAccountLocked      = errors.New("账户已被锁定，请稍后再试")
	ErrAccountDisabled    = errors.New("账户已被禁用")
	ErrTOTPRequired       = errors.New("需要双因素认证")
	ErrTOTPInvalid        = errors.New("双因素认证码无效")
	ErrTOTPNotEnabled     = errors.New("未启用双因素认证")
)

// HashPassword 对密码进行哈希
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePassword 验证密码强度
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("密码长度至少为8位")
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasDigit = true
		default:
			hasSpecial = true
		}
	}

	if !hasUpper {
		return errors.New("密码必须包含大写字母")
	}
	if !hasLower {
		return errors.New("密码必须包含小写字母")
	}
	if !hasDigit {
		return errors.New("密码必须包含数字")
	}
	if !hasSpecial {
		return errors.New("密码必须包含特殊字符")
	}

	return nil
}

// GenerateTOTPSecret 生成 TOTP 密钥
func GenerateTOTPSecret(username string) (string, string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "WebVMM",
		AccountName: username,
		Period:      30,
		SecretSize:  32,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", "", err
	}

	return key.Secret(), key.URL(), nil
}

// ValidateTOTP 验证 TOTP 代码
func ValidateTOTP(code, secret string) bool {
	return totp.Validate(code, secret)
}

// GenerateRandomString 生成随机字符串
func GenerateRandomString(length int) (string, error) {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		result[i] = chars[num.Int64()]
	}
	return string(result), nil
}

// CheckUserLock 检查用户是否被锁定
func CheckUserLock(user *models.User) error {
	if !user.IsActive {
		return ErrAccountDisabled
	}

	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		return ErrAccountLocked
	}

	return nil
}

// HandleFailedLogin 处理登录失败
func HandleFailedLogin(user *models.User, maxAttempts int, lockDuration time.Duration) error {
	user.FailedLogins++
	
	if user.FailedLogins >= maxAttempts {
		lockUntil := time.Now().Add(lockDuration)
		user.LockedUntil = &lockUntil
	}

	return nil
}

// ResetFailedLogins 重置登录失败计数
func ResetFailedLogins(user *models.User) {
	user.FailedLogins = 0
	user.LockedUntil = nil
}

// GenerateAPIKey 生成 API Key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// ConstantTimeCompare 常量时间比较
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// ParseBearerToken 从 Authorization header 解析 Bearer token
func ParseBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	
	return parts[1]
}

// GenerateRecoveryCodes 生成恢复代码
func GenerateRecoveryCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		code, err := GenerateRandomString(8)
		if err != nil {
			return nil, err
		}
		codes[i] = fmt.Sprintf("%s-%s", code[:4], code[4:])
	}
	return codes, nil
}