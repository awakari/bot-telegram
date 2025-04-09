package interests

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	apiGrpc "github.com/awakari/bot-telegram/api/grpc/interests"
	"github.com/awakari/bot-telegram/model"
	"github.com/awakari/bot-telegram/model/interest"
	"github.com/awakari/bot-telegram/model/interest/condition"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"net/http"
	"net/url"
)

type Service interface {
	Create(ctx context.Context, groupId, userId string, subData interest.Data) (id string, err error)
	Read(ctx context.Context, groupId, userId, subId string) (subData interest.Data, err error)
	Delete(ctx context.Context, groupId, userId, subId string) (err error)
	Search(ctx context.Context, groupId, userId string, q interest.Query, cursor condition.Cursor) (page []*apiGrpc.Interest, err error)
}

type service struct {
	clientHttp *http.Client
	url        string
	token      string
}

var ErrInternal = errors.New("internal failure")
var ErrNoAuth = errors.New("unauthenticated request")
var ErrInvalid = errors.New("invalid request")
var ErrLimitReached = errors.New("own interest limit reached")
var ErrNotFound = errors.New("interest not found")

var protoJsonUnmarshalOpts = protojson.UnmarshalOptions{
	DiscardUnknown: true,
	AllowPartial:   true,
}

func NewService(clientHttp *http.Client, url, token string) Service {
	return service{
		clientHttp: clientHttp,
		url:        url,
		token:      token,
	}
}

func (svc service) Create(ctx context.Context, groupId, userId string, subData interest.Data) (id string, err error) {

	reqProto := apiGrpc.CreateRequest{
		Description: subData.Description,
		Enabled:     subData.Enabled,
		Cond:        encodeCondition(subData.Condition),
		Public:      subData.Public,
	}
	if !subData.Expires.IsZero() {
		reqProto.Expires = timestamppb.New(subData.Expires)
	}

	var reqData []byte
	reqData, err = protojson.Marshal(&reqProto)

	var req *http.Request
	if err == nil {
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, svc.url, bytes.NewReader(reqData))
	}

	var resp *http.Response
	if err == nil {
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Authorization", "Bearer "+svc.token)
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add(model.KeyGroupId, groupId)
		req.Header.Add(model.KeyUserId, userId)
		resp, err = svc.clientHttp.Do(req)
	}

	if err == nil {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			err = ErrNoAuth
		case http.StatusBadRequest:
			err = ErrInvalid
		case http.StatusTooManyRequests:
			err = ErrLimitReached
		}
	}

	var respData []byte
	if err == nil {
		defer resp.Body.Close()
		respData, err = io.ReadAll(resp.Body)
	}

	var respProto apiGrpc.CreateResponse
	if err == nil {
		err = protoJsonUnmarshalOpts.Unmarshal(respData, &respProto)
	}

	if err == nil {
		id = respProto.Id
	}

	return
}

func (svc service) Read(ctx context.Context, groupId, userId, subId string) (subData interest.Data, err error) {

	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, svc.url+"/"+subId, nil)

	var resp *http.Response
	if err == nil {
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Authorization", "Bearer "+svc.token)
		req.Header.Add(model.KeyGroupId, groupId)
		req.Header.Add(model.KeyUserId, userId)
		resp, err = svc.clientHttp.Do(req)
	}

	if err == nil {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			err = ErrNoAuth
		case http.StatusNotFound:
			err = fmt.Errorf("%w: %s", ErrNotFound, subId)
		}
	}

	var respData []byte
	if err == nil {
		defer resp.Body.Close()
		respData, err = io.ReadAll(resp.Body)
	}

	var respProto apiGrpc.ReadResponse
	if err == nil {
		err = protoJsonUnmarshalOpts.Unmarshal(respData, &respProto)
	}

	if err == nil {
		subData.Condition, err = decodeCondition(respProto.Cond)
	}

	if err == nil {
		subData.Description = respProto.Description
		subData.Enabled = respProto.Enabled
		subData.Public = respProto.Public
		subData.Followers = respProto.Followers
		if respProto.Expires != nil && respProto.Expires.IsValid() {
			subData.Expires = respProto.Expires.AsTime()
		}
		if respProto.Created != nil && respProto.Created.IsValid() {
			subData.Created = respProto.Created.AsTime()
		}
		if respProto.Updated != nil && respProto.Updated.IsValid() {
			subData.Updated = respProto.Updated.AsTime()
		}
	}

	return
}

func (svc service) Delete(ctx context.Context, groupId, userId, subId string) (err error) {

	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, http.MethodDelete, svc.url+"/"+subId, nil)

	var resp *http.Response
	if err == nil {
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Authorization", "Bearer "+svc.token)
		req.Header.Add(model.KeyGroupId, groupId)
		req.Header.Add(model.KeyUserId, userId)
		resp, err = svc.clientHttp.Do(req)
	}

	if err == nil {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			err = ErrNoAuth
		case http.StatusNotFound:
			err = fmt.Errorf("%w: %s", ErrNotFound, subId)
		}
	}

	if err == nil {
		defer resp.Body.Close()
	}

	return
}

func (svc service) Search(ctx context.Context, groupId, userId string, q interest.Query, cursor condition.Cursor) (page []*apiGrpc.Interest, err error) {

	reqUrl := svc.url + "?sort="
	switch q.Sort {
	case interest.SortFollowers:
		reqUrl += "FOLLOWERS"
	default:
		reqUrl += "ID"
	}

	switch q.Order {
	case interest.OrderDesc:
		reqUrl += "&order=DESC"
	default:
		reqUrl += "&order=ASC"
	}

	if cursor.Id != "" {
		reqUrl += "&cursor=" + cursor.Id
	}

	if q.Public {
		reqUrl += fmt.Sprintf("&public=true&followers=%d", cursor.Followers)
	}
	if q.Limit > 0 {
		reqUrl += fmt.Sprintf("&limit=%d", q.Limit)
	}
	if q.Pattern != "" {
		reqUrl += "&filter=" + url.QueryEscape(q.Pattern)
	}

	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, reqUrl, nil)

	var resp *http.Response
	if err == nil {
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Authorization", "Bearer "+svc.token)
		req.Header.Add(model.KeyGroupId, groupId)
		req.Header.Add(model.KeyUserId, userId)
		resp, err = svc.clientHttp.Do(req)
	}

	var respData []byte
	if err == nil {
		defer resp.Body.Close()
		respData, err = io.ReadAll(resp.Body)
	}

	var respProto apiGrpc.SearchRestResponse
	if err == nil {
		err = protoJsonUnmarshalOpts.Unmarshal(respData, &respProto)
	}

	if err == nil {
		page = respProto.Subs
	}

	return
}

func encodeCondition(src condition.Condition) (dst *apiGrpc.Condition) {
	dst = &apiGrpc.Condition{
		Not: src.IsNot(),
	}
	switch c := src.(type) {
	case condition.GroupCondition:
		var dstGroup []*apiGrpc.Condition
		for _, childSrc := range c.GetGroup() {
			childDst := encodeCondition(childSrc)
			dstGroup = append(dstGroup, childDst)
		}
		dst.Cond = &apiGrpc.Condition_Gc{
			Gc: &apiGrpc.GroupCondition{
				Logic: apiGrpc.GroupLogic(c.GetLogic()),
				Group: dstGroup,
			},
		}
	case condition.TextCondition:
		dst.Cond = &apiGrpc.Condition_Tc{
			Tc: &apiGrpc.TextCondition{
				Key:   c.GetKey(),
				Term:  c.GetTerm(),
				Exact: c.IsExact(),
			},
		}
	case condition.SemanticCondition:
		dst.Cond = &apiGrpc.Condition_Sc{
			Sc: &apiGrpc.SemanticCondition{
				Query: c.Query(),
			},
		}
	case condition.NumberCondition:
		dstOp := encodeNumOp(c.GetOperation())
		dst.Cond = &apiGrpc.Condition_Nc{
			Nc: &apiGrpc.NumberCondition{
				Key: c.GetKey(),
				Op:  dstOp,
				Val: c.GetValue(),
			},
		}
	}
	return
}

func encodeNumOp(src condition.NumOp) (dst apiGrpc.Operation) {
	switch src {
	case condition.NumOpGt:
		dst = apiGrpc.Operation_Gt
	case condition.NumOpGte:
		dst = apiGrpc.Operation_Gte
	case condition.NumOpEq:
		dst = apiGrpc.Operation_Eq
	case condition.NumOpLte:
		dst = apiGrpc.Operation_Lte
	case condition.NumOpLt:
		dst = apiGrpc.Operation_Lt
	default:
		dst = apiGrpc.Operation_Undefined
	}
	return
}

func decodeCondition(src *apiGrpc.Condition) (dst condition.Condition, err error) {
	gc, nc, sc, tc := src.GetGc(), src.GetNc(), src.GetSc(), src.GetTc()
	switch {
	case gc != nil:
		var group []condition.Condition
		var childDst condition.Condition
		for _, childSrc := range gc.Group {
			childDst, err = decodeCondition(childSrc)
			if err != nil {
				break
			}
			group = append(group, childDst)
		}
		if err == nil {
			dst = condition.NewGroupCondition(
				condition.NewCondition(src.Not),
				condition.GroupLogic(gc.GetLogic()),
				group,
			)
		}
	case nc != nil:
		dstOp := decodeNumOp(nc.Op)
		dst = condition.NewNumberCondition(
			condition.NewKeyCondition(condition.NewCondition(src.Not), nc.GetKey()),
			dstOp, nc.Val,
		)
	case sc != nil:
		dst = condition.NewSemanticCondition(
			condition.NewCondition(src.Not),
			sc.GetId(), sc.GetQuery(),
		)
	case tc != nil:
		dst = condition.NewTextCondition(
			condition.NewKeyCondition(condition.NewCondition(src.Not), tc.GetKey()),
			tc.GetTerm(), tc.GetExact(),
		)
	default:
		err = fmt.Errorf("%w: unsupported condition type", ErrInternal)
	}
	return
}

func decodeNumOp(src apiGrpc.Operation) (dst condition.NumOp) {
	switch src {
	case apiGrpc.Operation_Gt:
		dst = condition.NumOpGt
	case apiGrpc.Operation_Gte:
		dst = condition.NumOpGte
	case apiGrpc.Operation_Eq:
		dst = condition.NumOpEq
	case apiGrpc.Operation_Lte:
		dst = condition.NumOpLte
	case apiGrpc.Operation_Lt:
		dst = condition.NumOpLt
	default:
		dst = condition.NumOpUndefined
	}
	return
}
