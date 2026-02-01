package alipay

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"example_shop/common/config"
)

var (
	// 网关地址（生产/沙箱）
	// - 生产：https://openapi.alipay.com/gateway.do
	// - 沙箱：https://openapi.alipaydev.com/gateway.do
	gatewayProd   = "https://openapi.alipay.com/gateway.do"
	gatewaySandbox = "https://openapi.alipaydev.com/gateway.do"
)

// gatewayURL 根据配置返回支付宝网关地址。
func gatewayURL() string {
	if config.Cfg.AliPay.IsProduction {
		return cleanConfigString(gatewayProd)
	}
	return cleanConfigString(gatewaySandbox)
}

func cleanConfigString(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "`")
	s = strings.Trim(s, "\"")
	return strings.TrimSpace(s)
}

// WapPayURL 发起“手机网站支付”（alipay.trade.wap.pay），返回支付宝收银台 URL。
//
// 参数：
// - outTradeNo：商户订单号（建议用本项目 pay_no，需全局唯一）
// - subject：订单标题
// - totalAmount：支付金额（单位：元）
//
// 配置：
// - NotifyURL/ReturnURL 优先读取 conf/config.yaml 的 AliPay.NotifyURL/AliPay.ReturnURL
// - 若未配置，会根据 Gateway 端口拼出 127.0.0.1 的默认值（仅适合本机自测）
func WapPayURL(outTradeNo, subject string, totalAmount float64) (string, error) {
	outTradeNo = strings.TrimSpace(outTradeNo)
	subject = strings.TrimSpace(subject)
	if outTradeNo == "" || subject == "" {
		return "", fmt.Errorf("out_trade_no/subject不能为空")
	}
	if totalAmount <= 0 {
		return "", fmt.Errorf("total_amount不合法")
	}

	cfg := config.Cfg.AliPay
	appID := cleanConfigString(cfg.AppId)
	if appID == "" {
		return "", fmt.Errorf("AliPay.AppId缺失")
	}
	privateKeyPEM := normalizeRSAKey(cleanConfigString(cfg.PrivateKey), "RSA PRIVATE KEY")
	priv, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return "", fmt.Errorf("解析AliPay.PrivateKey失败: %w", err)
	}

	// 1) 生成 notify_url / return_url
	notifyURL := cleanConfigString(cfg.NotifyURL)
	if notifyURL == "" && config.Cfg.Server.Gateway.Port != 0 {
		host := strings.TrimSpace(config.Cfg.Server.Gateway.Host)
		if host == "127.0.0.1" || strings.EqualFold(host, "localhost") {
			notifyURL = fmt.Sprintf("http://%s:%d/api/v1/pay/callback", host, config.Cfg.Server.Gateway.Port)
		}
	}
	returnURL := cleanConfigString(cfg.ReturnURL)
	if returnURL == "" && config.Cfg.Server.Gateway.Port != 0 {
		host := strings.TrimSpace(config.Cfg.Server.Gateway.Host)
		if host == "127.0.0.1" || strings.EqualFold(host, "localhost") {
			returnURL = fmt.Sprintf("http://%s:%d/", host, config.Cfg.Server.Gateway.Port)
		}
	}

	// 2) 构造 biz_content（JSON 字符串，属于支付宝签名参数的一部分）
	bizContentBytes, err := json.Marshal(map[string]string{
		"subject":      subject,
		"out_trade_no": outTradeNo,
		"total_amount": fmt.Sprintf("%.2f", totalAmount),
		"product_code": "QUICK_WAP_WAY",
	})
	if err != nil {
		return "", fmt.Errorf("构造biz_content失败: %w", err)
	}

	// 3) 构造签名参数（按支付宝开放平台规范）
	params := map[string]string{
		"app_id":    appID,
		"method":    "alipay.trade.wap.pay",
		"format":    "JSON",
		"charset":   "utf-8",
		"sign_type": "RSA2",
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"version":   "1.0",
		"biz_content": string(bizContentBytes),
	}
	if notifyURL != "" {
		params["notify_url"] = notifyURL
	}
	if returnURL != "" {
		params["return_url"] = returnURL
	}

	sign, err := signRSA2(params, priv)
	if err != nil {
		return "", err
	}

	v := url.Values{}
	for k, val := range params {
		v.Set(k, val)
	}
	v.Set("sign", sign)

	return gatewayURL() + "?" + v.Encode(), nil
}

func PagePayURL(outTradeNo, subject string, totalAmount float64) (string, error) {
	outTradeNo = strings.TrimSpace(outTradeNo)
	subject = strings.TrimSpace(subject)
	if outTradeNo == "" || subject == "" {
		return "", fmt.Errorf("out_trade_no/subject不能为空")
	}
	if totalAmount <= 0 {
		return "", fmt.Errorf("total_amount不合法")
	}

	cfg := config.Cfg.AliPay
	appID := cleanConfigString(cfg.AppId)
	if appID == "" {
		return "", fmt.Errorf("AliPay.AppId缺失")
	}
	privateKeyPEM := normalizeRSAKey(cleanConfigString(cfg.PrivateKey), "RSA PRIVATE KEY")
	priv, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return "", fmt.Errorf("解析AliPay.PrivateKey失败: %w", err)
	}

	notifyURL := cleanConfigString(cfg.NotifyURL)
	if notifyURL == "" && config.Cfg.Server.Gateway.Port != 0 {
		host := strings.TrimSpace(config.Cfg.Server.Gateway.Host)
		if host == "127.0.0.1" || strings.EqualFold(host, "localhost") {
			notifyURL = fmt.Sprintf("http://%s:%d/api/v1/pay/callback", host, config.Cfg.Server.Gateway.Port)
		}
	}
	returnURL := cleanConfigString(cfg.ReturnURL)
	if returnURL == "" && config.Cfg.Server.Gateway.Port != 0 {
		host := strings.TrimSpace(config.Cfg.Server.Gateway.Host)
		if host == "127.0.0.1" || strings.EqualFold(host, "localhost") {
			returnURL = fmt.Sprintf("http://%s:%d/", host, config.Cfg.Server.Gateway.Port)
		}
	}

	bizContentBytes, err := json.Marshal(map[string]string{
		"subject":      subject,
		"out_trade_no": outTradeNo,
		"total_amount": fmt.Sprintf("%.2f", totalAmount),
		"product_code": "FAST_INSTANT_TRADE_PAY",
	})
	if err != nil {
		return "", fmt.Errorf("构造biz_content失败: %w", err)
	}

	params := map[string]string{
		"app_id":      appID,
		"method":      "alipay.trade.page.pay",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": string(bizContentBytes),
	}
	if notifyURL != "" {
		params["notify_url"] = notifyURL
	}
	if returnURL != "" {
		params["return_url"] = returnURL
	}

	sign, err := signRSA2(params, priv)
	if err != nil {
		return "", err
	}

	v := url.Values{}
	for k, val := range params {
		v.Set(k, val)
	}
	v.Set("sign", sign)
	return gatewayURL() + "?" + v.Encode(), nil
}

// Verify 验证支付宝异步通知/同步回跳携带的签名。
//
// 规则：
// - 优先用 conf/config.yaml 的 AliPay.AliPublicKey 做本地 RSA2(SHA256WithRSA) 验签（推荐）
// - 若未配置 AliPublicKey，则回退走 SDK VerifySign（依赖 SDK 客户端内的 aliPublicKey）
func Verify(values url.Values) error {
	aliPublicKey := cleanConfigString(config.Cfg.AliPay.AliPublicKey)
	if aliPublicKey == "" {
		return fmt.Errorf("AliPay.AliPublicKey缺失，无法验签")
	}
	return verifyRSA2(values, aliPublicKey)
}

// normalizeRSAKey 把 “纯 base64 的 key” 自动包装成 PEM。
//
// 你配置文件里可能存的是一整段 PEM（包含 BEGIN/END），也可能只是 base64 内容；
// 这里统一规整成 PEM，便于后续解析。
func normalizeRSAKey(key, pemType string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if strings.Contains(key, "BEGIN") && strings.Contains(key, "END") {
		return key
	}

	key = strings.ReplaceAll(key, "\r", "")
	key = strings.ReplaceAll(key, "\n", "")
	key = strings.ReplaceAll(key, " ", "")
	if key == "" {
		return ""
	}

	var b strings.Builder
	b.WriteString("-----BEGIN " + pemType + "-----\n")
	for len(key) > 64 {
		b.WriteString(key[:64])
		b.WriteByte('\n')
		key = key[64:]
	}
	if len(key) > 0 {
		b.WriteString(key)
		b.WriteByte('\n')
	}
	b.WriteString("-----END " + pemType + "-----")
	return b.String()
}

// parseRSAPrivateKey 解析 RSA 私钥，支持 PKCS1 / PKCS8 两种格式。
func parseRSAPrivateKey(pemKey string) (*rsa.PrivateKey, error) {
	pemKey = strings.TrimSpace(pemKey)
	if pemKey == "" {
		return nil, fmt.Errorf("私钥为空")
	}
	block, _ := pem.Decode([]byte(pemKey))
	if block == nil || len(block.Bytes) == 0 {
		return nil, fmt.Errorf("私钥PEM解析失败")
	}
	if k, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return k, nil
	}
	any, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	k, ok := any.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("私钥类型不是RSA")
	}
	return k, nil
}

// signRSA2 对参数做 RSA2(SHA256WithRSA) 签名并返回 base64 字符串。
//
// 签名串规则：过滤 sign 字段（这里 params 本身不含 sign），按 key 升序拼成 k=v&k2=v2...
func signRSA2(params map[string]string, privateKey *rsa.PrivateKey) (string, error) {
	signData := buildSignDataFromMap(params)
	if signData == "" {
		return "", fmt.Errorf("签名数据为空")
	}
	sum := sha256.Sum256([]byte(signData))
	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, sum[:])
	if err != nil {
		return "", fmt.Errorf("RSA2签名失败: %w", err)
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func buildSignDataFromMap(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		if k == "sign" || k == "sign_type" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
	}
	return b.String()
}

// verifyRSA2 使用“支付宝公钥”做 RSA2(SHA256WithRSA) 验签。
//
// 核心规则：
// - 过滤 sign / sign_type
// - 剩余参数按 key 字典序排序，拼成 k=v&k2=v2...
// - 对拼接串做 SHA256
// - 用 RSA 公钥校验签名
func verifyRSA2(values url.Values, aliPublicKey string) error {
	sign := strings.TrimSpace(values.Get("sign"))
	if sign == "" {
		return fmt.Errorf("缺少sign")
	}
	sign = strings.ReplaceAll(sign, " ", "+")

	signBytes, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return fmt.Errorf("sign解码失败")
	}

	signData := buildSignData(values)
	if signData == "" {
		return fmt.Errorf("验签数据为空")
	}

	pub, err := parseRSAPublicKey(aliPublicKey)
	if err != nil {
		return err
	}

	sum := sha256.Sum256([]byte(signData))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], signBytes); err != nil {
		return fmt.Errorf("验签失败")
	}
	return nil
}

// buildSignData 按支付宝规则生成待验签字符串。
func buildSignData(values url.Values) string {
	keys := make([]string, 0, len(values))
	for k := range values {
		if k == "sign" || k == "sign_type" {
			continue
		}
		v := strings.TrimSpace(values.Get(k))
		if v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(values.Get(k))
	}
	return b.String()
}

// parseRSAPublicKey 解析支付宝公钥（支持 PKIX/PKCS1 两种格式）。
func parseRSAPublicKey(key string) (*rsa.PublicKey, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("AliPay.AliPublicKey缺失")
	}
	key = normalizeRSAKey(key, "PUBLIC KEY")
	block, _ := pem.Decode([]byte(key))
	if block == nil || len(block.Bytes) == 0 {
		return nil, fmt.Errorf("支付宝公钥解析失败")
	}

	if pubAny, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		if pub, ok := pubAny.(*rsa.PublicKey); ok {
			return pub, nil
		}
		return nil, fmt.Errorf("支付宝公钥类型不支持")
	}
	if pub, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return pub, nil
	}
	return nil, fmt.Errorf("支付宝公钥解析失败")
}
