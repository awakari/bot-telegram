package telegram

import "context"

type Service interface {
	Create(ctx context.Context, ch *Channel) (err error)
	Read(ctx context.Context, link string) (ch *Channel, err error)
	Delete(ctx context.Context, link string) (err error)
	List(ctx context.Context, filter *Filter, limit uint32, cursor string) (page []*Channel, err error)
}

type service struct {
	client ServiceClient
}

func NewService(client ServiceClient) Service {
	return service{
		client: client,
	}
}

func (svc service) Create(ctx context.Context, ch *Channel) (err error) {
	_, err = svc.client.Create(ctx, &CreateRequest{
		Channel: ch,
	})
	return
}

func (svc service) Read(ctx context.Context, link string) (ch *Channel, err error) {
	var resp *ReadResponse
	resp, err = svc.client.Read(ctx, &ReadRequest{
		Link: link,
	})
	if resp != nil {
		ch = resp.Channel
	}
	return
}

func (svc service) Delete(ctx context.Context, link string) (err error) {
	_, err = svc.client.Delete(ctx, &DeleteRequest{
		Link: link,
	})
	return
}

func (svc service) List(ctx context.Context, filter *Filter, limit uint32, cursor string) (page []*Channel, err error) {
	var resp *ListResponse
	resp, err = svc.client.List(ctx, &ListRequest{
		Filter: filter,
		Limit:  limit,
		Cursor: cursor,
	})
	if err == nil {
		page = resp.Page
	}
	return
}
