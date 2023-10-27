package feeds

import (
	"context"
)

type Service interface {

	// Create stores a new feed record.
	Create(ctx context.Context, feed *Feed) (err error)

	Read(ctx context.Context, url string) (feed *Feed, err error)

	Delete(ctx context.Context, url, userId string) (err error)

	List(ctx context.Context, filter *Filter, limit uint32, cursor string) (page []*Feed, err error)
}

type service struct {
	client ServiceClient
}

func NewService(client ServiceClient) Service {
	return service{
		client: client,
	}
}

func (svc service) Create(ctx context.Context, feed *Feed) (err error) {
	_, err = svc.client.Create(ctx, &CreateRequest{
		Feed: feed,
	})
	return
}

func (svc service) Read(ctx context.Context, url string) (feed *Feed, err error) {
	var resp *ReadResponse
	resp, err = svc.client.Read(ctx, &ReadRequest{
		Url: url,
	})
	if resp != nil {
		feed = resp.Feed
	}
	return
}

func (svc service) Delete(ctx context.Context, url, userId string) (err error) {
	_, err = svc.client.Delete(ctx, &DeleteRequest{
		Url:    url,
		UserId: userId,
	})
	return
}

func (svc service) List(ctx context.Context, filter *Filter, limit uint32, cursor string) (page []*Feed, err error) {
	var resp *ListResponse
	resp, err = svc.client.List(ctx, &ListRequest{
		Filter: filter,
		Limit:  limit,
		Cursor: cursor,
	})
	if resp != nil {
		page = resp.Page
	}
	return
}
