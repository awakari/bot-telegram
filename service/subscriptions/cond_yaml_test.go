package subscriptions

import (
	"encoding/json"
	"github.com/awakari/client-sdk-go/model/subscription/condition"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestConditionToYaml(t *testing.T) {
	cond := condition.
		NewBuilder().
		Any([]condition.Condition{
			condition.
				NewBuilder().
				Not().
				AttributeKey("key0").
				LessThanOrEqual(3.1415926).
				BuildNumberCondition(),
			condition.
				NewBuilder().
				AttributeKey("key1").
				TextEquals("text1").
				BuildTextCondition(),
		}).
		BuildGroupCondition()
	condJsonTxt := protojson.Format(encodeCondition(cond))
	m := map[string]interface{}{}
	err := json.Unmarshal([]byte(condJsonTxt), &m)
	require.Nil(t, err)
	var condYaml []byte
	condYaml, err = yaml.Marshal(m)
	require.Nil(t, err)
	assert.Equal(
		t,
		string(condYaml),
		`gc:
    group:
        - nc:
            key: key0
            op: Lte
            val: 3.1415926
          not: true
        - tc:
            exact: true
            key: key1
            term: text1
    logic: Or
`,
	)
}
