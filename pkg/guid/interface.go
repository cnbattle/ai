package guid

type Id interface {
	NextID() string
	WithPrefix(prefix string) string
}
