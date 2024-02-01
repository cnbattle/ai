package ai

import "cnbattle.com/ai/pkg/token"

var Token *token.Client

//TOKEN=true
//TOKEN_SECRET=aiio
//TOKEN_EXP=7200

func init() {
	if GetDefaultEnvToBool("TOKEN", false) {
		LOG.Trace("auto initialization TOKEN")
		Token = token.NewClient(GetDefaultEnv("TOKEN_SECRET", "aiio"),
			GetDefaultEnvToInt("TOKEN_EXP", 7200))
	}
}
