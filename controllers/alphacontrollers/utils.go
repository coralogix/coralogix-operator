package alphacontrollers

import (
	"time"

	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/utils/pointer"
)

const (
	defaultErrRequeuePeriod = 30 * time.Second
)

func WrapperspbStringToStringPointer(s *wrapperspb.StringValue) *string {
	if s == nil {
		return nil
	}
	return pointer.String(s.GetValue())
}
