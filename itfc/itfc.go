package itfc

type Cache interface {
	Find(fn func() (interface{}, error), prefix string, args ... interface{}) (v interface{}, err error)
}
