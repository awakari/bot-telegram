package feeds

import (
	"context"
	"google.golang.org/protobuf/types/known/durationpb"
	"time"
)

type Service interface {

	// Write stores a new feed record. It allows to overwrite the existing record only if it's expired.
	Write(ctx context.Context, feed *Feed) (err error)

	// Read returns the feed record by its unique url.URL. Returns zero value and ErrNotFound if missing.
	Read(ctx context.Context, url string) (feed *Feed, err error)

	// Extend adds the specified duration to the "Expires" field. Returns ErrNotFound if missing or expired already.
	Extend(ctx context.Context, url string, add time.Duration) (err error)

	List(ctx context.Context, filter *Filter, limit uint32, cursor string) (page []string, err error)
}

type service struct {
	client ServiceClient
}

func NewService(client ServiceClient) Service {
	return service{
		client: client,
	}
}

func (svc service) Write(ctx context.Context, feed *Feed) (err error) {
	_, err = svc.client.Write(ctx, &WriteRequest{
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

func (svc service) Extend(ctx context.Context, url string, add time.Duration) (err error) {
	_, err = svc.client.Extend(ctx, &ExtendRequest{
		Url: url,
		Add: durationpb.New(add),
	})
	return
}

func (svc service) List(ctx context.Context, filter *Filter, limit uint32, cursor string) (page []string, err error) {
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
