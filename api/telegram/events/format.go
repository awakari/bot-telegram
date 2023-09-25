package events

import (
	"fmt"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/microcosm-cc/bluemonday"
	"unicode/utf8"
)

const fmtLenMaxTitle = 128
const fmtLenMaxSummary = 512
const fmtLenMaxTxt = 512

type Format struct {
	HtmlPolicy *bluemonday.Policy
}

func (f Format) Html(evt *pb.CloudEvent) (txt string) {
	title, titleOk := evt.Attributes["title"]
	if titleOk {
		titleHtml := f.HtmlPolicy.Sanitize(title.GetCeString())
		txt += fmt.Sprintf("<b>%s</b>\n", truncateStringUtf8(titleHtml, fmtLenMaxTitle))
	}

	summary, summaryOk := evt.Attributes["summary"]
	if summaryOk {
		summaryHtml := f.HtmlPolicy.Sanitize(summary.GetCeString())
		txt += fmt.Sprintf("%s\n", truncateStringUtf8(summaryHtml, fmtLenMaxSummary))
	}

	txtData := evt.GetTextData()
	switch {
	case txtData != "":
		txtHtml := f.HtmlPolicy.Sanitize(txtData)
		txt += fmt.Sprintf("%s\n", truncateStringUtf8(txtHtml, fmtLenMaxTxt))
	}

	urlImg, urlImgOk := evt.Attributes["imageurl"]
	if !urlImgOk {
		urlImg, urlImgOk = evt.Attributes["feedimageurl"]
	}
	if urlImgOk {
		switch {
		case urlImg.GetCeString() != "":
			txt += fmt.Sprintf("<a href=\"%s\" alt=\"image\">  </a>\n", urlImg.GetCeString())
		case urlImg.GetCeUri() != "":
			txt += fmt.Sprintf("<a href=\"%s\" alt=\"image\">  </a>\n", urlImg.GetCeUri())
		}
	}

	urlSrc := evt.Source
	rssItemGuid, rssItemGuidOk := evt.Attributes["rssitemguid"]
	if rssItemGuidOk {
		urlSrc = rssItemGuid.GetCeString()
	}
	txt += fmt.Sprintf("<a href=\"%s\">Read more</a>\n", urlSrc)

	groupIdSrc, groupIdSrcOk := evt.Attributes["awakarigroupid"]
	if groupIdSrcOk {
		txt += fmt.Sprintf("From: %s\n", groupIdSrc.GetCeString())
	}

	return
}

func truncateStringUtf8(s string, lenMax int) string {
	if len(s) <= lenMax {
		return s
	}
	// Ensure we don't split a UTF-8 character in the middle.
	for i := lenMax - 3; i > 0; i-- {
		if utf8.RuneStart(s[i]) {
			return s[:i] + "..."
		}
	}
	return ""
}
