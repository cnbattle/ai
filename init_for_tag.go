package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cnbattle.com/ai/pkg/cache"
	"cnbattle.com/ai/pkg/guid"
	"cnbattle.com/ai/pkg/sms"
	"cnbattle.com/ai/pkg/token"
	"gorm.io/gorm"
)

// env name
// DB_TAG_DSN=root:123456@(127.0.0.1:3306)/aiio?charset=utf8mb4&parseTime=true&loc=Local
// DB_TAG_PREFIX=

func InitGormForTag(tag string) *gorm.DB {
	return InitGorm(GetEnv(fmt.Sprintf("DB_%v_DSN", strings.ToUpper(tag))), fmt.Sprintf("DB_%v_PREFIX", strings.ToUpper(tag)))
}

// env name
// CACHE_TAG_PROVIDER=Redis or FreeCache or BigCache
// CACHE_TAG_HOST=127.0.0.1:6379
// CACHE_TAG_PASS=123456
// CACHE_TAG_DB=1
// CACHE_TAG_EXT=10

func InitCacheForTag(tag string) cache.Cache {
	Cache, err = cache.NewClient(GetEnv(fmt.Sprintf("CACHE_%v_PROVIDER", strings.ToUpper(tag))),
		GetEnv(fmt.Sprintf("CACHE_%v_HOST", strings.ToUpper(tag))),
		GetEnv(fmt.Sprintf("CACHE_%v_PASS", strings.ToUpper(tag))),
		GetDefaultEnvToInt(fmt.Sprintf("CACHE_%v_DB", strings.ToUpper(tag)), 1),
		GetDefaultEnvToInt(fmt.Sprintf("CACHE_%v_EXT", strings.ToUpper(tag)), 10),
		context.Background(),
	)
	if err != nil {
		panic(fmt.Sprintf("InitCache err:%v", err))
	}
	return Cache
}

// env name
// GUID_TAG_START_TIME=2022-12-12T12:12:12+08:00
// GUID_TAG_ENGINE=idgen
// GUID_TAG_WORKER_ID=1

func InitGUIDForTag(tag string) guid.Id {
	parse, err := time.Parse(time.RFC3339, GetDefaultEnv(fmt.Sprintf("GUID_%v_START_TIME ", strings.ToUpper(tag)), "2022-12-12T12:12:12+08:00"))
	if err != nil {
		panic(fmt.Sprintf("InitGUID err:%v", err))
	}
	GUID, err = guid.New(GetDefaultEnv(fmt.Sprintf("GUID_%v_ENGINE", strings.ToUpper(tag)), "idgen"),
		parse,
		uint16(GetDefaultEnvToInt(fmt.Sprintf("GUID_%v_WORKER_ID", strings.ToUpper(tag)), 1)),
	)
	if err != nil {
		panic(fmt.Sprintf("InitGUID err:%v", err))
	}
	return GUID
}

// env name
// TOKEN_TAG_SECRET=aiio
// TOKEN_TAG_EXP=7200

func InitTokenForTag(tag string) *token.Client {
	return token.NewClient(GetDefaultEnv(fmt.Sprintf("TOKEN_%v_SECRET", strings.ToUpper(tag)), "aiio"),
		GetDefaultEnvToInt(fmt.Sprintf("TOKEN_%v_EXP", strings.ToUpper(tag)), 7200))
}

// env name
// SMS_TAG_PROVIDER=TencentCloudSMS
// SMS_TAG_ACCESS_ID=AKIDylIZZIk24xxxx
// SMS_TAG_ACCESS_KEY=WmkRr0tVptgBPN4Tbxxxx
// SMS_TAG_APP_ID=1400620000
// SMS_TAG_SIGN=短信签名
// SMS_TAG_TEMPLATE=1270000

func InitSMSForTag(tag string) sms.Client {
	SMS, err := sms.NewClient(GetEnv(fmt.Sprintf("SMS_%v_PROVIDER", strings.ToUpper(tag))),
		GetEnv(fmt.Sprintf("SMS_%v_ACCESS_ID", strings.ToUpper(tag))),
		GetEnv(fmt.Sprintf("SMS_%v_ACCESS_KEY", strings.ToUpper(tag))),
		GetEnv(fmt.Sprintf("SMS_%v_SIGN", strings.ToUpper(tag))),
		GetEnv(fmt.Sprintf("SMS_%v_TEMPLATE", strings.ToUpper(tag))),
		GetEnv(fmt.Sprintf("SMS_%v_APP_ID", strings.ToUpper(tag))))
	if err != nil {
		panic(fmt.Sprintf("InitSMS err:%v", err))
	}
	return SMS
}
