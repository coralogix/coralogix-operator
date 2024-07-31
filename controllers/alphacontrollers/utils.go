package alphacontrollers

import (
	"time"

	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/utils/pointer"
)

const (
	defaultRequeuePeriod    = 30 * time.Minute
	defaultErrRequeuePeriod = 1 * time.Minute
)

func WrapperspbStringToStringPointer(s *wrapperspb.StringValue) *string {
	if s == nil {
		return nil
	}
	return pointer.String(s.GetValue())
}
