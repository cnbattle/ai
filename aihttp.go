package ai

import (
	"cnbattle.com/ai/pkg/aihttp"
	restyv2 "github.com/go-resty/resty/v2"
	restyv3 "resty.dev/v3"
)

func RestyV2New() *restyv2.Client {
	return aihttp.RestyV2New()
}

func RestyV3New() *restyv3.Client {
	return aihttp.RestyV3New()
}
