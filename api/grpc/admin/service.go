package admin

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/client-sdk-go/api/grpc/subject"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

type Service interface {
	SetLimits(ctx context.Context, groupId, userId string, subj subject.Subject, count int64, expires time.Time) (err error)
}

type service struct {
	client ServiceClient
}

var ErrInternal = errors.New("internal failure")

var ErrInvalidArg = errors.New("invalid argument")

func NewService(client ServiceClient) Service {
	return service{
		client: client,
	}
}

func (svc service) SetLimits(ctx context.Context, groupId, userId string, subj subject.Subject, count int64, expires time.Time) (err error) {
	req := SetLimitsRequest{
		GroupId: groupId,
		UserId:  userId,
		Subj:    Subject(subj),
		Count:   count,
		Expires: timestamppb.New(expires.UTC()),
	}
	_, err = svc.client.SetLimits(ctx, &req)
	err = decodeError(err)
	return
}

func decodeError(src error) (dst error) {
	switch status.Code(src) {
	case codes.OK:
		dst = nil
	case codes.InvalidArgument:
		dst = fmt.Errorf("%w: %s", ErrInvalidArg, src)
	default:
		dst = fmt.Errorf("%w: %s", ErrInternal, src)
	}
	return
}
