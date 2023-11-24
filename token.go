package ai

var Token *TokenClient

//TOKEN=true
//TOKEN_SECRET=aiio
//TOKEN_EXP=7200

func init() {
	if GetDefaultEnvToBool("TOKEN", false) {
		Token = NewTokenClient(GetDefaultEnv("TOKEN_SECRET", "cnbattle"),
			GetDefaultEnvToInt("TOKEN_EXP", 7200))
	}
}
