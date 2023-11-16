package subscriptions

import (
	"github.com/awakari/client-sdk-go/api/grpc/subscriptions"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConditionHandler_Update(t *testing.T) {
	cases := map[string]struct {
		in  string
		err error
		out *subscriptions.Condition
	}{
		"ok": {
			in: `{"not":false,"gc":{"logic":1,"group":[{"not":true,"nc":{"key":"key0","val":-3.1415926,"op":2}},{"not":false,"tc":{"key":"title","term":"awakari","exact":true}}]}}`,
			out: &subscriptions.Condition{
				Not: false,
				Cond: &subscriptions.Condition_Gc{
					Gc: &subscriptions.GroupCondition{
						Logic: subscriptions.GroupLogic_Or,
						Group: []*subscriptions.Condition{
							{
								Cond: &subscriptions.Condition_Nc{
									Nc: &subscriptions.NumberCondition{
										Key: "key0",
										Op:  subscriptions.Operation_Gte,
										Val: -3.1415926,
									},
								},
							},
							{
								Cond: &subscriptions.Condition_Tc{
									Tc: &subscriptions.TextCondition{
										Key:   "title",
										Term:  "awakari",
										Exact: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			out := &subscriptions.Condition{}
			err := convertConditionJsonToProto([]byte(c.in), out)
			assert.ErrorIs(t, err, c.err)
			assert.Equal(t, c.out.Not, out.Not)
		})
	}
}
