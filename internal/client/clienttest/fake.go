package clienttest

import (
	"context"
	"encoding/json"
	"errors"
)

type FakeCaller struct {
	Responses map[string]json.RawMessage
	Errors    map[string]error
}

func (f *FakeCaller) Call(_ context.Context, method string, _ any) (json.RawMessage, error) {
	if err, ok := f.Errors[method]; ok {
		return nil, err
	}
	if result, ok := f.Responses[method]; ok {
		return result, nil
	}
	return nil, errors.New("FakeCaller: no response configured for method " + method)
}

func (f *FakeCaller) CallWithJob(_ context.Context, _ string, _ any) (json.RawMessage, error) {
	return nil, errors.New("CallWithJob not implemented")
}
