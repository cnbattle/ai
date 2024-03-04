package ai

import (
	"fmt"
	"time"

	"cnbattle.com/ai/pkg/guid"
)

var GUID guid.Id

//GUID=true
//GUID_START_TIME=2022-12-12T12:12:12+08:00
//GUID_ENGINE=idgen
//GUID_WORKER_ID=1

func init() {
	if GetDefaultEnvToBool("GUID", false) {
		LOG.Debug("auto initialization GUID")
		parse, err := time.Parse(time.RFC3339, GetDefaultEnv("GUID_START_TIME", "2022-12-12T12:12:12+08:00"))
		if err != nil {
			panic(err)
		}
		GUID, err = guid.New(GetDefaultEnv("GUID_ENGINE", "idgen"),
			parse,
			uint16(GetDefaultEnvToInt("GUID_WORKER_ID", 1)),
		)
		if err != nil {
			panic(fmt.Sprintf("InitGUID err:%v", err))
		}
	}
}
