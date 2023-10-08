package messages

import (
	"fmt"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/microcosm-cc/bluemonday"
	"gopkg.in/telebot.v3"
	"unicode/utf8"
)

const fmtLenMaxTitle = 128
const fmtLenMaxSummary = 256
const fmtLenMaxTxt = 256

type Format struct {
	HtmlPolicy *bluemonday.Policy
}

func (f Format) Convert(evt *pb.CloudEvent) (tgMsg any) {
	fileTypeAttr, fileTypeFound := evt.Attributes[attrKeyFileType]
	switch fileTypeFound {
	case true:
		ft := FileType(fileTypeAttr.GetCeInteger())
		file := telebot.File{
			FileID:   evt.Attributes[attrKeyFileId].GetCeString(),
			UniqueID: evt.Attributes[attrKeyFileUniqueId].GetCeString(),
		}
		switch ft {
		case FileTypeAudio:
			tgMsg = &telebot.Audio{
				File:     file,
				Duration: int(evt.Attributes[attrKeyFileMediaDuration].GetCeInteger()),
				Caption:  f.Html(evt),
			}
		case FileTypeDocument:
			tgMsg = &telebot.Document{
				File:    file,
				Caption: f.Html(evt),
			}
		case FileTypeImage:
			tgMsg = &telebot.Photo{
				File:    file,
				Width:   int(evt.Attributes[attrKeyFileImgWidth].GetCeInteger()),
				Height:  int(evt.Attributes[attrKeyFileImgHeight].GetCeInteger()),
				Caption: f.Html(evt),
			}
		case FileTypeVideo:
			tgMsg = &telebot.Video{
				File:     file,
				Width:    int(evt.Attributes[attrKeyFileImgWidth].GetCeInteger()),
				Height:   int(evt.Attributes[attrKeyFileImgHeight].GetCeInteger()),
				Duration: int(evt.Attributes[attrKeyFileMediaDuration].GetCeInteger()),
				Caption:  f.Html(evt),
			}
		}
	default:
		tgMsg = f.Html(evt)
	}
	return
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
		txt += fmt.Sprintf("\n%s\n", truncateStringUtf8(summaryHtml, fmtLenMaxSummary))
	}

	txtData := evt.GetTextData()
	switch {
	case txtData != "":
		txtHtml := f.HtmlPolicy.Sanitize(txtData)
		txt += fmt.Sprintf("\n%s\n", truncateStringUtf8(txtHtml, fmtLenMaxTxt))
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

	groupIdSrc, groupIdSrcOk := evt.Attributes["awakarigroupid"]
	if groupIdSrcOk {
		txt += fmt.Sprintf("\nVia: %s\n", groupIdSrc.GetCeString())
	}

	urlSrc := evt.Source
	rssItemGuid, rssItemGuidOk := evt.Attributes["rssitemguid"]
	if rssItemGuidOk {
		urlSrc = rssItemGuid.GetCeString()
	}
	txt += fmt.Sprintf("<a href=\"%s\">Source</a>\n", urlSrc)

	txt += fmt.Sprintf("\nSincerely yours,\n@AwakariBot")

	return
}

func (f Format) Plain(evt *pb.CloudEvent) (txt string) {
	title, titleOk := evt.Attributes["title"]
	if titleOk {
		titleHtml := f.HtmlPolicy.Sanitize(title.GetCeString())
		txt += fmt.Sprintf("%s\n", truncateStringUtf8(titleHtml, fmtLenMaxTitle))
	}

	summary, summaryOk := evt.Attributes["summary"]
	if summaryOk {
		summaryHtml := f.HtmlPolicy.Sanitize(summary.GetCeString())
		txt += fmt.Sprintf("\n%s\n", truncateStringUtf8(summaryHtml, fmtLenMaxSummary))
	}

	txtData := evt.GetTextData()
	switch {
	case txtData != "":
		txtHtml := f.HtmlPolicy.Sanitize(txtData)
		txt += fmt.Sprintf("\n%s\n", truncateStringUtf8(txtHtml, fmtLenMaxTxt))
	}

	groupIdSrc, groupIdSrcOk := evt.Attributes["awakarigroupid"]
	if groupIdSrcOk {
		txt += fmt.Sprintf("\nVia: %s\n", groupIdSrc.GetCeString())
	}

	urlSrc := evt.Source
	rssItemGuid, rssItemGuidOk := evt.Attributes["rssitemguid"]
	if rssItemGuidOk {
		urlSrc = rssItemGuid.GetCeString()
	}
	txt += fmt.Sprintf("Source: %s", urlSrc)

	txt += fmt.Sprintf("\nSincerely yours,\n@AwakariBot")

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
