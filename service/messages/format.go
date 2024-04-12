package messages

import (
	"encoding/base64"
	"fmt"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/microcosm-cc/bluemonday"
	"gopkg.in/telebot.v3"
	"strings"
	"time"
	"unicode/utf8"
)

const fmtLenMaxAttrVal = 256
const fmtLenMaxBodyTxt = 1024

type Format struct {
	HtmlPolicy *bluemonday.Policy
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
	switch attrSummaryFound {
	case true:
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
	default:
		txtData := evt.GetTextData()
		switch {
		case txtData != "":
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
	if obj != "" {
		txt += obj + "\n\n"
	}
	//
	subDetailsLink := "https://awakari.com/sub-details.html?id=" + subId
	if mode == FormatModeHtml {
		subDetailsLink = "<a href=\"" + subDetailsLink + "\">" + subDescr + "</a>"
	}
	txt += "Subscription: " + subDetailsLink + "\n\n"
	//
	var attrsTxt string
	if attrs {
		attrsTxt = f.convertExtraAttrs(evt, mode, trunc)
	}
	if attrsTxt != "" {
		switch mode {
		case FormatModeHtml:
			txt += fmt.Sprintf("<span class=\"tg-spoiler\">%s</span>\n", attrsTxt)
		default:
			txt += fmt.Sprintf("%s\n", attrsTxt)
		}
	}
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

func (f Format) convertExtraAttrs(evt *pb.CloudEvent, mode FormatMode, trunc bool) (txt string) {
	txt += fmt.Sprintf("id: %s\nsource: %s\ntype: %s\n", evt.Id, evt.Source, evt.Type)
	for attrName, attrVal := range evt.Attributes {
		switch attrName {
		case "summary":
		case "title":
		case "awakarimatchfound": // internal
		case "awakariuserid": // do not expose
		case "awkhash": // internal, useless
		case "awkinternal": // internal
		case "srccategories":
		case "srcdescription":
		case "srcimagetitle":
		case "srcimageurl":
		case "srctitle":
		default:
			switch vt := attrVal.Attr.(type) {
			case *pb.CloudEventAttributeValue_CeBoolean:
				switch vt.CeBoolean {
				case true:
					txt += fmt.Sprintf("%s: true\n", attrName)
				default:
					txt += fmt.Sprintf("%s: false\n", attrName)
				}
			case *pb.CloudEventAttributeValue_CeInteger:
				txt += fmt.Sprintf("%s: %d\n", attrName, vt.CeInteger)
			case *pb.CloudEventAttributeValue_CeString:
				if vt.CeString != evt.Source { // "object"/"objecturl" might the same value as the source
					v := f.HtmlPolicy.Sanitize(vt.CeString)
					if trunc {
						v = truncateStringUtf8(v, fmtLenMaxAttrVal)
					}
					txt += fmt.Sprintf("%s: %s\n", attrName, v)
				}
			case *pb.CloudEventAttributeValue_CeUri:
				v := vt.CeUri
				if trunc {
					v = truncateStringUtf8(v, fmtLenMaxAttrVal)
				}
				txt += fmt.Sprintf("%s: %s\n", attrName, v)
			case *pb.CloudEventAttributeValue_CeUriRef:
				v := vt.CeUriRef
				if trunc {
					v = truncateStringUtf8(v, fmtLenMaxAttrVal)
				}
				txt += fmt.Sprintf("%s: %s\n", attrName, v)
			case *pb.CloudEventAttributeValue_CeTimestamp:
				v := vt.CeTimestamp
				txt += fmt.Sprintf("%s: %s\n", attrName, v.AsTime().Format(time.RFC3339))
			case *pb.CloudEventAttributeValue_CeBytes:
				v := base64.StdEncoding.EncodeToString(vt.CeBytes)
				if trunc {
					v = truncateStringUtf8(v, fmtLenMaxAttrVal)
				}
				txt += fmt.Sprintf("%s: %s\n", attrName, v)
			}
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
