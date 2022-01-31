package gorpc

import (
	"reflect"
	"testing"
)

type TestService struct {
}

func (*TestService) method1() {
	return
}

func (*TestService) Method2(args *string, reply *string) error {
	return nil
}

func Test_getServiceMethods(t *testing.T) {
	typeof := reflect.TypeOf(new(TestService))
	methods := getServiceMethods(typeof)
	t.Logf("methods:%+v", methods)
}
