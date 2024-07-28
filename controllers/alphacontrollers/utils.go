package alphacontrollers

import (
	"time"

	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/utils/pointer"
)

const (
	defaultRequeuePeriod    = 30 * time.Second
	defaultErrRequeuePeriod = 20 * time.Second
)

func WrapperspbStringToStringPointer(s *wrapperspb.StringValue) *string {
	if s == nil {
		return nil
	}
	return pointer.String(s.GetValue())
}
