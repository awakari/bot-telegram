package messages

import (
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/microcosm-cc/bluemonday"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
	"testing"
	"time"
)

func TestFormat_Convert(t *testing.T) {
	// init events format, see https://core.telegram.org/bots/api#html-style for details
	htmlPolicy := bluemonday.NewPolicy()
	htmlPolicy.AllowStandardURLs()
	htmlPolicy.
		AllowAttrs("href").
		OnElements("a")
	htmlPolicy.AllowElements("b", "strong", "i", "em", "u", "ins", "s", "strike", "del", "code", "pre")
	htmlPolicy.
		AllowAttrs("class").
		OnElements("span")
	htmlPolicy.AllowURLSchemes("tg")
	htmlPolicy.
		AllowAttrs("emoji-ids").
		OnElements("tg-emoji")
	htmlPolicy.
		AllowAttrs("class").
		OnElements("code")
	htmlPolicy.AllowDataURIImages()
	fmtMsg := Format{
		HtmlPolicy: htmlPolicy,
	}
	cases := map[string]struct {
		in  *pb.CloudEvent
		out any
	}{
		"1": {
			in: &pb.CloudEvent{
				Id:          "82f39262-5eb4-4f7f-9142-7c489d670907",
				Source:      "https://bbs.archlinux.org/extern.php?action=feed&fid=32&type=atom",
				SpecVersion: "1.0",
				Type:        "com.awakari.api.permits.exhausted",
				Attributes: map[string]*pb.CloudEventAttributeValue{
					"time": {
						Attr: &pb.CloudEventAttributeValue_CeTimestamp{
							CeTimestamp: timestamppb.New(time.Date(2024, 5, 31, 23, 54, 00, 0, time.UTC)),
						},
					},
				},
				Data: &pb.CloudEvent_TextData{
					TextData: `⚠ Daily publishing limit reached.

Increase your daily publication limit or nominate own sources for the dedicated limit.

If you did not publish messages, <a href="https://awakari.com/pub.html?own=true">check own publication sources</a> you added.`,
				},
			},
			out: "⚠ Daily publishing limit reached.\n\nIncrease your daily publication limit or nominate own source...\n\n<a href=\"https://bbs.archlinux.org/extern.php?action=feed&fid=32&type=atom\">https://bbs.archlinux.org/extern.php?action=feed&fid=32&type=atom</a>\n\nInterest: <a href=\"https://awakari.com/sub-details.html?id=sub1\">sub1 description</a>\n\n<a href=\"82f39262-5eb4-4f7f-9142-7c489d670907&interestId=sub1\">Result Details</a>",
		},
	}
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			out := fmtMsg.Convert(c.in, "sub1", "sub1 description", FormatModeHtml)
			assert.Equal(t, c.out, out)
		})
	}
}
