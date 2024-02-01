package token

// Claims JWT Claims
type Claims struct {
	Exp  int    `json:"exp"`
	Iat  int    `json:"iat"`
	UID  string `json:"uid"`
	Role string `json:"role"`
}
