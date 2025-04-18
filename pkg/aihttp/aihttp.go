package aihttp

import (
	restyv2 "github.com/go-resty/resty/v2"
	restyv3 "resty.dev/v3"
)

func RestyV2New() *restyv2.Client {
	return restyv2.New()
}

func RestyV3New() *restyv3.Client {
	return restyv3.New()
}
