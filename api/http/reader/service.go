package reader

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/bytedance/sonic"
	"io"
	"net/http"
	"time"
)

type Service interface {
	CreateCallback(ctx context.Context, subId, url string, interval time.Duration) (err error)
	GetCallback(ctx context.Context, subId, url string) (cb Callback, err error)
	DeleteCallback(ctx context.Context, subId, url string) (err error)
	ListByUrl(ctx context.Context, limit uint32, url, cursor string) (page []string, err error)
}

type service struct {
	clientHttp *http.Client
	uriBase    string
}

const keyHubCallback = "hub.callback"
const KeyHubMode = "hub.mode"
const KeyHubTopic = "hub.topic"
const modeSubscribe = "subscribe"
const modeUnsubscribe = "unsubscribe"
const fmtTopicUri = "%s/sub/%s/%s"
const FmtJson = "json"

var ErrInternal = errors.New("internal failure")
var ErrConflict = errors.New("conflict")
var ErrNotFound = errors.New("not found")

func NewService(clientHttp *http.Client, uriBase string) Service {
	return service{
		clientHttp: clientHttp,
		uriBase:    uriBase,
	}
}

func (svc service) CreateCallback(ctx context.Context, subId, callbackUrl string, interval time.Duration) (err error) {
	err = svc.updateCallback(ctx, subId, callbackUrl, modeSubscribe, interval)
	return
}

func (svc service) GetCallback(ctx context.Context, subId, url string) (cb Callback, err error) {
	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/callbacks/%s/%s", svc.uriBase, subId, base64.URLEncoding.EncodeToString([]byte(url))), http.NoBody)
	var resp *http.Response
	if err == nil {
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
			err = fmt.Errorf("%w: response status %d", ErrInternal, resp.StatusCode)
		}
	default:
		err = fmt.Errorf("%w: %s", ErrInternal, err)
	}
	return
}

func (svc service) DeleteCallback(ctx context.Context, subId, callbackUrl string) (err error) {
	err = svc.updateCallback(ctx, subId, callbackUrl, modeUnsubscribe, 0)
	return
}

func (svc service) ListByUrl(ctx context.Context, limit uint32, url, cursor string) (page []string, err error) {
	var req *http.Request
	req, err = http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf(
			"%s/callbacks/list-by-url/%s?cursor=%s&limit=%d",
			svc.uriBase,
			base64.URLEncoding.EncodeToString([]byte(url)),
			cursor,
			limit,
		),
		http.NoBody,
	)
	var resp *http.Response
	if err == nil {
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
			err = fmt.Errorf("%w: response status %d", ErrInternal, resp.StatusCode)
		}
	default:
		err = fmt.Errorf("%w: %s", ErrInternal, err)
	}
	if err == nil {
		page = ip.Page
	}
	return
}

func (svc service) updateCallback(_ context.Context, subId, url, mode string, interval time.Duration) (err error) {
	topicUri := fmt.Sprintf(fmtTopicUri, svc.uriBase, FmtJson, subId)
	data := map[string][]string{
		keyHubCallback: {
			url,
		},
		KeyHubMode: {
			mode,
		},
		KeyHubTopic: {
			topicUri,
		},
	}
	reqUri := topicUri
	if interval > 0 {
		reqUri += "?interval=" + interval.String()
	}
	var resp *http.Response
	resp, err = svc.clientHttp.PostForm(reqUri, data)
	switch err {
	case nil:
		switch resp.StatusCode {
		case http.StatusAccepted, http.StatusNoContent:
		case http.StatusNotFound:
			err = fmt.Errorf("%w: callback not found for the subscription %s", ErrConflict, subId)
		case http.StatusConflict:
			err = fmt.Errorf("%w: callback already registered for the subscription %s", ErrConflict, subId)
		default:
			defer resp.Body.Close()
			respBody, _ := io.ReadAll(resp.Body)
			err = fmt.Errorf("%w: unexpected create callback response %d, %s", ErrInternal, resp.StatusCode, string(respBody))
		}
	default:
		err = fmt.Errorf("%w: %s", ErrInternal, err)
	}
	return
}
