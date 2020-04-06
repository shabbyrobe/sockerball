package exampleproto

type OK struct{}

type TestRequest struct {
	Foo string
}

type TestResponse struct {
	Bar string
}

type TestCommand struct{}
