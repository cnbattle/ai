package ai

import (
	"fmt"

	"cnbattle.com/ai/pkg/sms"
)

var SMS sms.Client

//SMS=true
//SMS_PROVIDER=TencentCloudSMS
//SMS_ACCESS_ID=AKIDylIZZIk24xxxx
//SMS_ACCESS_KEY=WmkRr0tVptgBPN4Tbxxxx
//SMS_APP_ID=1400620000
//SMS_SIGN=短信签名
//SMS_TEMPLATE=1270000

func init() {
	var err error
	if GetDefaultEnvToBool("SMS", false) {
		LOG.Trace("auto initialization SMS")
		SMS, err = sms.NewClient(GetEnv("SMS_PROVIDER"),
			GetEnv("SMS_ACCESS_ID"),
			GetEnv("SMS_ACCESS_KEY"),
			GetEnv("SMS_SIGN"),
			GetEnv("SMS_TEMPLATE"),
			GetEnv("SMS_APP_ID"))
		if err != nil {
			panic(fmt.Sprintf("InitSMS err:%v", err))
		}
	}
}
