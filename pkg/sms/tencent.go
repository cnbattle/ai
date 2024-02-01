/*
 * SPDX-License-Identifier: Apache-2.0 License
 * Author: cnbattle  <qiaicn@gmail.com>
 * Copyright (c) 2022.
 */

package sms

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type SendSmsRequest struct {
	// 下发手机号码，采用 E.164 标准，格式为+[国家或地区码][手机号]，单次请求最多支持200个手机号且要求全为境内手机号或全为境外手机号。
	// 例如：+8613711112222， 其中前面有一个+号 ，86为国家码，13711112222为手机号。
	PhoneNumberSet []string `json:"PhoneNumberSet,omitempty"`

	// 短信 SdkAppId，在 [短信控制台](https://console.cloud.tencent.com/smsv2/app-manage)  添加应用后生成的实际 SdkAppId，示例如1400006666。
	SmsSdkAppId string `json:"SmsSdkAppId,omitempty"`

	// 模板 ID，必须填写已审核通过的模板 ID。模板 ID 可登录 [短信控制台](https://console.cloud.tencent.com/smsv2) 查看，若向境外手机号发送短信，仅支持使用国际/港澳台短信模板。
	TemplateId string `json:"TemplateId,omitempty"`

	// 短信签名内容，使用 UTF-8 编码，必须填写已审核通过的签名，例如：腾讯云，签名信息可登录 [短信控制台](https://console.cloud.tencent.com/smsv2)  查看。
	// 注：国内短信为必填参数。
	SignName string `json:"SignName,omitempty"`

	// 模板参数，若无模板参数，则设置为空。
	TemplateParamSet []string `json:"TemplateParamSet,omitempty"`

	// 短信码号扩展号，默认未开通，如需开通请联系 [sms helper](https://cloud.tencent.com/document/product/382/3773#.E6.8A.80.E6.9C.AF.E4.BA.A4.E6.B5.81)。
	ExtendCode string `json:"ExtendCode,omitempty"`

	// 用户的 session 内容，可以携带用户侧 ID 等上下文信息，server 会原样返回。
	SessionContext string `json:"SessionContext,omitempty"`

	// 国内短信无需填写该项；国际/港澳台短信已申请独立 SenderId 需要填写该字段，默认使用公共 SenderId，无需填写该字段。
	// 注：月度使用量达到指定量级可申请独立 SenderId 使用，详情请联系 [sms helper](https://cloud.tencent.com/document/product/382/3773#.E6.8A.80.E6.9C.AF.E4.BA.A4.E6.B5.81)。
	SenderId string `json:"SenderId,omitempty"`
}

type SendSmsResponse struct {
	Response struct {
		SendStatusSet []struct {
			SerialNo       string `json:"SerialNo"`
			PhoneNumber    string `json:"PhoneNumber"`
			Fee            int    `json:"Fee"`
			SessionContext string `json:"SessionContext"`
			Code           string `json:"Code"`
			Message        string `json:"Message"`
			IsoCode        string `json:"IsoCode"`
		} `json:"SendStatusSet"`
		RequestID string `json:"RequestId"`
	} `json:"Response"`
}

type TencentClient struct {
	accessId  string
	accessKey string

	appId    string
	sign     string
	template string
}

func NewTencentClient(accessId string, accessKey string, sign string, templateId string, appId string) (*TencentClient, error) {
	tencentClient := &TencentClient{
		accessId:  accessId,
		accessKey: accessKey,
		appId:     appId,
		sign:      sign,
		template:  templateId,
	}

	return tencentClient, nil
}

func (c *TencentClient) SendMessage(param map[string]string, targetPhoneNumber ...string) error {
	var paramArray []string
	index := 0
	for {
		value := param[strconv.Itoa(index)]
		if len(value) == 0 {
			break
		}
		paramArray = append(paramArray, value)
		index++
	}
	for i, s := range targetPhoneNumber {
		targetPhoneNumber[i] = "+86" + s
	}
	return c.send(paramArray, targetPhoneNumber...)
}

func (c *TencentClient) send(param []string, targetPhoneNumber ...string) error {
	host := "sms.tencentcloudapi.com"
	algorithm := "TC3-HMAC-SHA256"
	service := "sms"
	version := "2021-01-11"
	action := "SendSms"
	region := "ap-guangzhou"
	var timestamp = time.Now().Unix()

	// step 1: build canonical request string
	httpRequestMethod := "POST"
	canonicalURI := "/"
	canonicalQueryString := ""

	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-tc-action:%s\n",
		"application/json; charset=utf-8", host, strings.ToLower(action))
	signedHeaders := "content-type;host;x-tc-action"
	request := SendSmsRequest{
		SmsSdkAppId:      c.appId,
		SignName:         c.sign,
		TemplateId:       c.template,
		TemplateParamSet: param,
		PhoneNumberSet:   targetPhoneNumber,
	}
	payload, _ := json.Marshal(request)
	hashedRequestPayload := sha256hex(string(payload))
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		httpRequestMethod,
		canonicalURI,
		canonicalQueryString,
		canonicalHeaders,
		signedHeaders,
		hashedRequestPayload)

	// step 2: build string to sign
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, service)
	hashedCanonicalRequest := sha256hex(canonicalRequest)
	string2sign := fmt.Sprintf("%s\n%d\n%s\n%s",
		algorithm,
		timestamp,
		credentialScope,
		hashedCanonicalRequest)

	// step 3: sign string
	secretDate := hmacsha256(date, "TC3"+c.accessKey)
	secretService := hmacsha256(service, secretDate)
	secretSigning := hmacsha256("tc3_request", secretService)
	signature := hex.EncodeToString([]byte(hmacsha256(string2sign, secretSigning)))

	// step 4: build authorization
	authorization := fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm,
		c.accessId,
		credentialScope,
		signedHeaders,
		signature)

	client := &http.Client{}
	var data = strings.NewReader(string(payload))
	req, err := http.NewRequest("POST", "https://sms.tencentcloudapi.com", data)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Host", "sms.tencentcloudapi.com")
	req.Header.Set("X-TC-Action", action)
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-TC-Version", version)
	req.Header.Set("X-TC-Region", region)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var respData SendSmsResponse
	err = json.Unmarshal(bodyText, &respData)
	if err != nil {
		return err
	}

	if strings.Contains(respData.Response.SendStatusSet[0].Code, "Ok") {
		return nil
	}
	return errors.New(respData.Response.SendStatusSet[0].Message)
}

func sha256hex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

func hmacsha256(s, key string) string {
	hashed := hmac.New(sha256.New, []byte(key))
	hashed.Write([]byte(s))
	return string(hashed.Sum(nil))
}
