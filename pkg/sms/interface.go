package sms

type Client interface {
	SendMessage(param map[string]string, targetPhoneNumber ...string) error
}
