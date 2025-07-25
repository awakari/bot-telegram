package reader

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/model"
	"github.com/bytedance/sonic"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Service interface {
	Subscribe(ctx context.Context, interestId, groupId, userId, url string, interval time.Duration) (err error)
	Subscription(ctx context.Context, interestId, groupId, userId, url string) (cb Subscription, err error)
	Unsubscribe(ctx context.Context, interestId, groupId, userId, url string) (err error)
	InterestsByUrl(ctx context.Context, groupId, userId string, limit uint32, url, cursor string) (page []string, err error)
}

type service struct {
	clientHttp    *http.Client
	uriBase       string
	tokenInternal string
}

const keyHubCallback = "hub.callback"
const KeyHubMode = "hub.mode"
const KeyHubTopic = "hub.topic"
const modeSubscribe = "subscribe"
const modeUnsubscribe = "unsubscribe"
const fmtTopicUri = "%s/v1/sub/%s/%s"
const FmtJson = "json"

var ErrInternal = errors.New("internal failure")
var ErrConflict = errors.New("conflict")
var ErrNotFound = errors.New("not found")
var ErrPermitExhausted = errors.New("permit exhausted")

func NewService(clientHttp *http.Client, uriBase, tokenInternal string) Service {
	return service{
		clientHttp:    clientHttp,
		uriBase:       uriBase,
		tokenInternal: tokenInternal,
	}
}

func (svc service) Subscribe(ctx context.Context, interestId, groupId, userId, urlCallback string, interval time.Duration) (err error) {
	err = svc.updateCallback(ctx, interestId, groupId, userId, urlCallback, modeSubscribe, interval)
	return
}

func (svc service) Subscription(ctx context.Context, interestId, groupId, userId, urlCallback string) (cb Subscription, err error) {
	var req *http.Request
	req, err = http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/v2?interestId=%s&url=%s", svc.uriBase, interestId, base64.URLEncoding.EncodeToString([]byte(urlCallback))),
		http.NoBody,
	)
	var resp *http.Response
	if err == nil {
		req.Header.Set("Authorization", "Bearer "+svc.tokenInternal)
		req.Header.Set(model.KeyGroupId, groupId)
		req.Header.Set(model.KeyUserId, userId)
		resp, err = svc.clientHttp.Do(req)
	}
	switch err {
	case nil:
		defer resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusOK:
			err = sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&cb)
			if err != nil {
				err = fmt.Errorf("%w: %s", ErrInternal, err)
			}
		case http.StatusNotFound:
			err = ErrNotFound
		default:
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 0x1000))
			err = fmt.Errorf("%w: response %d, %s", ErrInternal, resp.StatusCode, string(body))
		}
	default:
		err = fmt.Errorf("%w: %s", ErrInternal, err)
	}
	return
}

func (svc service) Unsubscribe(ctx context.Context, interestId, groupId, userId, urlCallback string) (err error) {
	err = svc.updateCallback(ctx, interestId, groupId, userId, urlCallback, modeUnsubscribe, 0)
	return
}

func (svc service) InterestsByUrl(ctx context.Context, groupId, userId string, limit uint32, urlCallback, cursor string) (page []string, err error) {
	var req *http.Request
	req, err = http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf(
			"%s/v2?url=%s&cursor=%s&limit=%d",
			svc.uriBase,
			base64.URLEncoding.EncodeToString([]byte(urlCallback)),
			cursor,
			limit,
		),
		http.NoBody,
	)
	var resp *http.Response
	if err == nil {
		req.Header.Set("Authorization", "Bearer "+svc.tokenInternal)
		req.Header.Set(model.KeyGroupId, groupId)
		req.Header.Set(model.KeyUserId, userId)
		resp, err = svc.clientHttp.Do(req)
	}
	var ip interestPage
	switch err {
	case nil:
		defer resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusOK:
			err = sonic.ConfigDefault.NewDecoder(resp.Body).Decode(&ip)
			if err != nil {
				err = fmt.Errorf("%w: %s", ErrInternal, err)
			}
		case http.StatusNotFound:
			err = ErrNotFound
		default:
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 0x1000))
			err = fmt.Errorf("%w: response %d, %s", ErrInternal, resp.StatusCode, string(body))
		}
	default:
		err = fmt.Errorf("%w: %s", ErrInternal, err)
	}
	if err == nil {
		page = ip.Page
	}
	return
}

func (svc service) updateCallback(ctx context.Context, interestId, groupId, userId, urlCallback, mode string, interval time.Duration) (err error) {

	topicUri := fmt.Sprintf(fmtTopicUri, svc.uriBase, FmtJson, interestId)
	data := url.Values{
		keyHubCallback: {
			urlCallback,
		},
		KeyHubMode: {
			mode,
		},
		KeyHubTopic: {
			topicUri,
		},
	}
	reqUri := fmt.Sprintf("%s/v2?format=%s&interestId=%s", svc.uriBase, FmtJson, interestId)
	if interval > 0 && mode == modeSubscribe {
		reqUri += "&interval=" + interval.String()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqUri, strings.NewReader(data.Encode()))
	var resp *http.Response
	if err == nil {
		req.Header.Set("Authorization", "Bearer "+svc.tokenInternal)
		req.Header.Set(model.KeyGroupId, groupId)
		req.Header.Set(model.KeyUserId, userId)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err = svc.clientHttp.Do(req)
	}

	switch err {
	case nil:
		switch resp.StatusCode {
		case http.StatusAccepted, http.StatusNoContent:
		case http.StatusNotFound:
			err = fmt.Errorf("%w: callback not found for the subscription %s", ErrConflict, interestId)
		case http.StatusConflict:
			err = fmt.Errorf("%w: callback already registered for the subscription %s", ErrConflict, interestId)
		case http.StatusTooManyRequests:
			err = ErrPermitExhausted
		default:
			defer resp.Body.Close()
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 0x1000))
			err = fmt.Errorf("%w: unexpected create callback response %d, %s", ErrInternal, resp.StatusCode, string(body))
		}
	default:
		err = fmt.Errorf("%w: %s", ErrInternal, err)
	}
	return
}
