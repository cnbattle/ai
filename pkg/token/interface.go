package token

type Interface interface {
	GenerateToken(uid, role string) (string, error)
	VerifyToken(tokenStr string) (claims Claims, err error)
	VerifyTokenForRole(tokenStr, role string) (claims Claims, err error)
	RenewToken(tokenStr string) (string, error)
}
