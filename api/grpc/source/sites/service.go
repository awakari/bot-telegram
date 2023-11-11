package sites

import (
	"context"
)

type Service interface {

	// Create stores a new feed record.
	Create(ctx context.Context, site *Site) (err error)

	Read(ctx context.Context, addr string) (site *Site, err error)

	Delete(ctx context.Context, addr, groupId, userId string) (err error)

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

func (svc service) Create(ctx context.Context, site *Site) (err error) {
	_, err = svc.client.Create(ctx, &CreateRequest{
		Site: site,
	})
	return
}

func (svc service) Read(ctx context.Context, addr string) (feed *Site, err error) {
	var resp *ReadResponse
	resp, err = svc.client.Read(ctx, &ReadRequest{
		Addr: addr,
	})
	if resp != nil {
		feed = resp.Site
	}
	return
}

func (svc service) Delete(ctx context.Context, url, groupId, userId string) (err error) {
	_, err = svc.client.Delete(ctx, &DeleteRequest{
		Addr:    url,
		GroupId: groupId,
		UserId:  userId,
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
