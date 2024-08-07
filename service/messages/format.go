package messages

import (
	"fmt"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/microcosm-cc/bluemonday"
	"gopkg.in/telebot.v3"
	"strings"
	"unicode/utf8"
)

const fmtLenMaxAttrVal = 100
const fmtLenMaxBodyTxt = 100

type Format struct {
	HtmlPolicy       *bluemonday.Policy
	UriReaderEvtBase string
}

type FormatMode int

const (
	FormatModeHtml  FormatMode = iota
	FormatModePlain            // no html markup, but keep the telegram attachments
	FormatModeRaw              // no html and no attachments
)

var htmlStripTags = bluemonday.
	StrictPolicy().
	AddSpaceWhenStrippingTag(true)

func (f Format) Convert(evt *pb.CloudEvent, subId, subDescr string, mode FormatMode) (tgMsg any) {
	fileTypeAttr, fileTypeFound := evt.Attributes[attrKeyFileType]
	if fileTypeFound && mode != FormatModeRaw {
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
				Caption:  f.convert(evt, subId, subDescr, mode, false, false),
			}
		case FileTypeDocument:
			tgMsg = &telebot.Document{
				File:    file,
				Caption: f.convert(evt, subId, subDescr, mode, false, false),
			}
		case FileTypeImage:
			tgMsg = &telebot.Photo{
				File:    file,
				Width:   int(evt.Attributes[attrKeyFileImgWidth].GetCeInteger()),
				Height:  int(evt.Attributes[attrKeyFileImgHeight].GetCeInteger()),
				Caption: f.convert(evt, subId, subDescr, mode, false, false),
			}
		case FileTypeVideo:
			tgMsg = &telebot.Video{
				File:     file,
				Width:    int(evt.Attributes[attrKeyFileImgWidth].GetCeInteger()),
				Height:   int(evt.Attributes[attrKeyFileImgHeight].GetCeInteger()),
				Duration: int(evt.Attributes[attrKeyFileMediaDuration].GetCeInteger()),
				Caption:  f.convert(evt, subId, subDescr, mode, false, false),
			}
		}
	} else {
		_, msgFromTg := evt.Attributes[attrKeyMsgId]
		switch msgFromTg {
		case true:
			// no need to truncate for telegram when message is from telegram
			// no need to convert any other attributes except text and footer
			tgMsg = f.convert(evt, subId, subDescr, mode, false, true)
		default:
			tgMsg = f.convert(evt, subId, subDescr, mode, true, true)
		}
	}
	return
}

func (f Format) convert(evt *pb.CloudEvent, subId, subDescr string, mode FormatMode, trunc, attrs bool) (txt string) {
	if attrs {
		txt += f.convertHeaderAttrs(evt, mode, trunc)
	}
	attrSummary, attrSummaryFound := evt.Attributes["summary"]
	if attrSummaryFound {
		v := attrSummary.GetCeString()
		switch mode {
		case FormatModeHtml:
			v = f.HtmlPolicy.Sanitize(v)
		default:
			v = htmlStripTags.Sanitize(v)
		}
		if trunc {
			v = truncateStringUtf8(v, fmtLenMaxBodyTxt)
		}
		txt += fmt.Sprintf("%s\n\n", v)
	}
	txtData := evt.GetTextData()
	if txtData != "" {
		switch mode {
		case FormatModeHtml:
			txtData = f.HtmlPolicy.Sanitize(txtData)
		default:
			txtData = htmlStripTags.Sanitize(txtData)
		}
		if trunc {
			txtData = truncateStringUtf8(txtData, fmtLenMaxBodyTxt)
		}
		txt += fmt.Sprintf("%s\n\n", txtData)
	}
	attrName, attrNameFound := evt.Attributes["name"]
	if txt == "" && attrNameFound {
		txt = fmt.Sprintf("%s\n\n", attrName.GetCeString())
	}
	//
	objAttr, objAttrFound := evt.Attributes["object"]
	var obj string
	if objAttrFound {
		switch objAttr.Attr.(type) {
		case *pb.CloudEventAttributeValue_CeString:
			obj = objAttr.GetCeString()
		case *pb.CloudEventAttributeValue_CeUri:
			obj = objAttr.GetCeUri()
		}
	}
	if obj == "" || (!strings.HasPrefix(obj, "https://") && !strings.HasPrefix(obj, "http://")) {
		objAttr, objAttrFound = evt.Attributes["objecturl"]
		if objAttrFound {
			switch objAttr.Attr.(type) {
			case *pb.CloudEventAttributeValue_CeString:
				obj = objAttr.GetCeString()
			case *pb.CloudEventAttributeValue_CeUri:
				obj = objAttr.GetCeUri()
			}
		}
	}
	if obj == "" {
		obj = evt.Source
	}
	//
	addrEvtAttrs := f.UriReaderEvtBase + evt.Id
	addrInterest := "https://awakari.com/sub-details.html?id=" + subId
	switch mode {
	case FormatModeHtml:
		txt += "Original: <a href=\"" + obj + "\">" + obj + "</a>\n\n"
		txt += "Interest: <a href=\"" + addrInterest + "\">" + subDescr + "</a>\n\n"
		txt += "<a href=\"" + addrEvtAttrs + "\">All Event Attributes</a>"
	default:
		txt += "Original: " + obj + "\n\nInterest: " + addrInterest + "\n\nAll Event Attributes: " + addrEvtAttrs
	}
	//
	return
}

func (f Format) convertHeaderAttrs(evt *pb.CloudEvent, mode FormatMode, trunc bool) (txt string) {
	attrTitle, attrTitleFound := evt.Attributes["title"]
	if attrTitleFound {
		v := f.HtmlPolicy.Sanitize(attrTitle.GetCeString())
		if trunc {
			v = truncateStringUtf8(v, fmtLenMaxAttrVal)
		}
		switch mode {
		case FormatModeHtml:
			txt += fmt.Sprintf("<b>%s</b>\n\n", v)
		default:
			txt += fmt.Sprintf("%s\n\n", v)
		}
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
