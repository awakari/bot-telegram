package telegram

import "context"

type Service interface {
	List(ctx context.Context, limit uint32, cursor string) (page []*Channel, err error)
}

type service struct {
	client ServiceClient
}

func NewService(client ServiceClient) Service {
	return service{
		client: client,
	}
}

func (svc service) List(ctx context.Context, limit uint32, cursor string) (page []*Channel, err error) {
	var resp *ListResponse
	resp, err = svc.client.List(ctx, &ListRequest{
		Limit:  limit,
		Cursor: cursor,
	})
	if err == nil {
		page = resp.Page
	}
	return
}
