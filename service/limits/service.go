package limits

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/api/grpc/auth"
	"github.com/awakari/bot-telegram/api/grpc/usage/limits"
	apiGrpcUsageSubject "github.com/awakari/bot-telegram/api/grpc/usage/subject"
	"github.com/awakari/bot-telegram/model/usage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"time"
)

type Service interface {
	Get(ctx context.Context, groupId, userId string, subj usage.Subject) (l usage.Limit, err error)
	Set(ctx context.Context, groupId, userId string, subj usage.Subject, count int64, expires time.Time) (err error)
	Delete(ctx context.Context, groupId, userId string, subjs ...usage.Subject) (err error)
}

type service struct {
	client limits.ServiceClient
}

var ErrInternal = errors.New("internal failure")
var ErrInvalid = errors.New("invalid")
var ErrNotFound = errors.New("not found")
var ErrForbidden = errors.New("forbidden")

func NewService(
	client limits.ServiceClient,
) Service {
	return service{
		client: client,
	}
}

func (svc service) Get(ctx context.Context, groupId, userId string, subj usage.Subject) (l usage.Limit, err error) {
	req := limits.GetRequest{
		Raw: false,
	}
	var resp *limits.GetResponse
	req.Subj, err = apiGrpcUsageSubject.Encode(subj)
	if err == nil {
		ctxAuth := auth.SetOutgoingAuthInfo(ctx, groupId, userId)
		resp, err = svc.client.Get(ctxAuth, &req)
	}
	if err == nil {
		l.Count = resp.Count
		l.UserId = resp.UserId
		if resp.Expires != nil {
			l.Expires = resp.Expires.AsTime()
		}
	}
	err = decodeError(err)
	return
}

func (svc service) Set(ctx context.Context, groupId, userId string, subj usage.Subject, count int64, expires time.Time) (err error) {
	req := limits.SetRequest{
		Count:   count,
		UserId:  userId,
		GroupId: groupId,
	}
	req.Subj, err = apiGrpcUsageSubject.Encode(subj)
	if !expires.IsZero() {
		req.Expires = timestamppb.New(expires.UTC())
	}
	_, err = svc.client.Set(ctx, &req)
	err = decodeError(err)
	return
}

func (svc service) Delete(ctx context.Context, groupId, userId string, subjs ...usage.Subject) (err error) {
	req := limits.DeleteRequest{
		GroupId: groupId,
		UserId:  userId,
	}
	for _, s := range subjs {
		var reqSubj apiGrpcUsageSubject.Subject
		reqSubj, err = apiGrpcUsageSubject.Encode(s)
		if err == nil {
			req.Subjs = append(req.Subjs, reqSubj)
		}
	}
	if err == nil {
		_, err = svc.client.Delete(ctx, &req)
	}
	err = decodeError(err)
	return
}

func decodeError(src error) (dst error) {
	switch {
	case src == io.EOF:
		dst = src // return as it is
	case status.Code(src) == codes.OK:
		dst = nil
	case status.Code(src) == codes.InvalidArgument:
		dst = fmt.Errorf("%w: %s", ErrInvalid, src)
	case status.Code(src) == codes.NotFound:
		dst = fmt.Errorf("%w: %s", ErrNotFound, src)
	case status.Code(src) == codes.Unauthenticated:
		dst = fmt.Errorf("%w: %s", ErrForbidden, src)
	default:
		dst = fmt.Errorf("%w: %s", ErrInternal, src)
	}
	return
}
