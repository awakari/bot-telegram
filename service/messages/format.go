package messages

import (
	"encoding/base64"
	"fmt"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/microcosm-cc/bluemonday"
	"gopkg.in/telebot.v3"
	"time"
	"unicode/utf8"
)

const fmtLenMaxAttrVal = 256
const fmtLenMaxSummary = 512
const fmtLenMaxBodyTxt = 1024

type Format struct {
	HtmlPolicy *bluemonday.Policy
}

type FormatMode int

const (
	FormatModeHtml FormatMode = iota
	FormatModePlain
	FormatModeRaw
)

func (f Format) Convert(evt *pb.CloudEvent, mode FormatMode) (tgMsg any) {
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
				Caption:  f.convert(evt, mode, false, false),
			}
		case FileTypeDocument:
			tgMsg = &telebot.Document{
				File:    file,
				Caption: f.convert(evt, mode, false, false),
			}
		case FileTypeImage:
			tgMsg = &telebot.Photo{
				File:    file,
				Width:   int(evt.Attributes[attrKeyFileImgWidth].GetCeInteger()),
				Height:  int(evt.Attributes[attrKeyFileImgHeight].GetCeInteger()),
				Caption: f.convert(evt, mode, false, false),
			}
		case FileTypeVideo:
			tgMsg = &telebot.Video{
				File:     file,
				Width:    int(evt.Attributes[attrKeyFileImgWidth].GetCeInteger()),
				Height:   int(evt.Attributes[attrKeyFileImgHeight].GetCeInteger()),
				Duration: int(evt.Attributes[attrKeyFileMediaDuration].GetCeInteger()),
				Caption:  f.convert(evt, mode, false, false),
			}
		}
	} else {
		_, msgFromTg := evt.Attributes[attrKeyMsgId]
		switch msgFromTg {
		case true:
			// no need to truncate for telegram when message is from telegram
			// no need to convert any other attributes except text and footer
			tgMsg = f.convert(evt, mode, false, false)
		default:
			tgMsg = f.convert(evt, mode, true, true)
		}
	}
	return
}

func (f Format) convert(evt *pb.CloudEvent, mode FormatMode, trunc, attrs bool) (txt string) {

	if attrs {
		txt += f.convertHeaderAttrs(evt, mode, trunc)
	}

	txtData := evt.GetTextData()
	switch {
	case txtData != "":
		txtData = f.HtmlPolicy.Sanitize(txtData)
		if trunc {
			txtData = truncateStringUtf8(txtData, fmtLenMaxBodyTxt)
		}
		txt += fmt.Sprintf("%s\n", txtData)
	}

	urlSrc := evt.Source
	rssItemGuid, rssItemGuidOk := evt.Attributes["rssitemguid"]
	if rssItemGuidOk {
		urlSrc = rssItemGuid.GetCeString()
	}
	txt += fmt.Sprintf("Source: %s\n\n", urlSrc)

	var attrsTxt string
	if attrs {
		attrsTxt = f.convertAttrs(evt, mode, trunc)
	}
	if attrsTxt != "" {
		txt += fmt.Sprintf("%s\n", f.convertAttrs(evt, mode, trunc))
	}

	groupIdSrc, groupIdSrcOk := evt.Attributes["awakarigroupid"]
	if groupIdSrcOk {
		txt += fmt.Sprintf("Submitted by: %s\n", groupIdSrc.GetCeString())
	}
	txt += fmt.Sprintf("Delivered by: @AwakariBot")

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
	attrSummary, attrSummaryFound := evt.Attributes["summary"]
	if attrSummaryFound {
		v := f.HtmlPolicy.Sanitize(attrSummary.GetCeString())
		if trunc {
			v = truncateStringUtf8(v, fmtLenMaxSummary)
		}
		txt += fmt.Sprintf("%s\n\n", v)
	}
	return
}

func (f Format) convertAttrs(evt *pb.CloudEvent, mode FormatMode, trunc bool) (txt string) {

	for attrName, attrVal := range evt.Attributes {
		switch attrName {
		case "title":
		case "summary":
		case "awakarimatchfound": // internal
		case "awakarigroupid": // already in use for the "Via"
		case "awakariuserid": // do not expose
		case "rssitemguid": // already in use for the source "Source"
		case "feedcategories":
		case "feeddescription":
		case "feedimagetitle":
		case "feedimageurl":
		case "feedtitle":
		case "imagetitle": // already used when handling "imageurl" attr
		case "imageurl":
			var imgTitle string
			attrImgTitle, imgTitleFound := evt.Attributes["imagetitle"]
			switch imgTitleFound {
			case true:
				imgTitle = attrImgTitle.GetCeString()
			default:
				imgTitle = "Image"
			}
			switch mode {
			case FormatModeHtml:
				switch {
				case attrVal.GetCeString() != "":
					txt += fmt.Sprintf("<a href=\"%s\" alt=\"image\">%s</a>\n", imgTitle, attrVal.GetCeString())
				case attrVal.GetCeUri() != "":
					txt += fmt.Sprintf("<a href=\"%s\" alt=\"image\">%s</a>\n", imgTitle, attrVal.GetCeUri())
				}
			default:
				switch {
				case attrVal.GetCeString() != "":
					txt += fmt.Sprintf("%s: %s\n", imgTitle, attrVal.GetCeString())
				case attrVal.GetCeUri() != "":
					txt += fmt.Sprintf("%s: %s\n", imgTitle, attrVal.GetCeUri())
				}
			}
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
				v := f.HtmlPolicy.Sanitize(vt.CeString)
				if trunc {
					v = truncateStringUtf8(v, fmtLenMaxAttrVal)
				}
				txt += fmt.Sprintf("%s: %s\n", attrName, v)
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
