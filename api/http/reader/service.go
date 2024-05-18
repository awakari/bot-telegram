package reader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type Service interface {
	CreateCallback(ctx context.Context, subId, url string) (err error)
	GetCallback(ctx context.Context, subId string) (cb Callback, err error)
	DeleteCallback(ctx context.Context, subId, url string) (err error)
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

func (svc service) CreateCallback(ctx context.Context, subId, callbackUrl string) (err error) {
	err = svc.updateCallback(ctx, subId, callbackUrl, modeSubscribe)
	return
}

func (svc service) GetCallback(ctx context.Context, subId string) (cb Callback, err error) {
	var resp *http.Response
	resp, err = svc.clientHttp.Get(fmt.Sprintf("/callbacks/%s/%s", svc.uriBase, subId))
	switch err {
	case nil:
		defer resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusOK:
			err = json.NewDecoder(resp.Body).Decode(&cb)
			if err != nil {
				err = fmt.Errorf("%w: %s", ErrInternal, err)
			}
		case http.StatusNotFound:
			err = ErrNotFound
		default:
			err = fmt.Errorf("%w: response status %d", ErrInternal, resp.StatusCode)
		}
		err = json.NewDecoder(resp.Body).Decode(&cb)
	default:
		err = fmt.Errorf("%w: %s", ErrInternal, err)
	}
	return
}

func (svc service) DeleteCallback(ctx context.Context, subId, callbackUrl string) (err error) {
	err = svc.updateCallback(ctx, subId, callbackUrl, modeUnsubscribe)
	return
}

func (svc service) updateCallback(_ context.Context, subId, url, mode string) (err error) {
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
	var resp *http.Response
	resp, err = svc.clientHttp.PostForm(topicUri, data)
	switch err {
	case nil:
		switch resp.StatusCode {
		case http.StatusAccepted, http.StatusNoContent:
		case http.StatusNotFound:
			err = fmt.Errorf("%w: callback not found for the subscription %s", ErrConflict, subId)
		case http.StatusConflict:
			err = fmt.Errorf("%w: callback already registered for the subscription %s", ErrConflict, subId)
		default:
			err = fmt.Errorf("%w: unexpected create callback response %d", ErrInternal, resp.StatusCode)
		}
	default:
		err = fmt.Errorf("%w: %s", ErrInternal, err)
	}
	return
}
