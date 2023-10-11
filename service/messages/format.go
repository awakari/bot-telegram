package messages

import (
	"encoding/base64"
	"fmt"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/microcosm-cc/bluemonday"
	"gopkg.in/telebot.v3"
	"net/url"
	"time"
	"unicode/utf8"
)

const fmtLenMaxBodyTxt = 1024
const fmtLenMaxAttrVal = 256

type Format struct {
	HtmlPolicy *bluemonday.Policy
}

func (f Format) Convert(evt *pb.CloudEvent, html bool) (tgMsg any) {
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
				Caption:  f.convertTextDataAndFooter(evt, html, false),
			}
		case FileTypeDocument:
			tgMsg = &telebot.Document{
				File:    file,
				Caption: f.convertTextDataAndFooter(evt, html, false),
			}
		case FileTypeImage:
			tgMsg = &telebot.Photo{
				File:    file,
				Width:   int(evt.Attributes[attrKeyFileImgWidth].GetCeInteger()),
				Height:  int(evt.Attributes[attrKeyFileImgHeight].GetCeInteger()),
				Caption: f.convertTextDataAndFooter(evt, html, false),
			}
		case FileTypeVideo:
			tgMsg = &telebot.Video{
				File:     file,
				Width:    int(evt.Attributes[attrKeyFileImgWidth].GetCeInteger()),
				Height:   int(evt.Attributes[attrKeyFileImgHeight].GetCeInteger()),
				Duration: int(evt.Attributes[attrKeyFileMediaDuration].GetCeInteger()),
				Caption:  f.convertTextDataAndFooter(evt, html, false),
			}
		}
	default:
		_, msgFromTg := evt.Attributes[attrKeyMsgId]
		switch msgFromTg {
		case true:
			// no need to convert any other attributes except text and footer
			// no need to truncate for telegram if message came from telegram
			tgMsg = f.convertTextDataAndFooter(evt, html, false)
		default:
			tgMsg = f.convert(evt, html, true)
		}
	}
	return
}

func (f Format) convert(evt *pb.CloudEvent, html bool, trunc bool) (txt string) {

	for attrName, attrVal := range evt.Attributes {
		switch attrName {
		case "awakarigroupid": // already in use for the "Via"
		case "awakariuserid": // do not expose
		case "rssitemguid": // already in use for the source "Source"
		case "feedcategories":
		case "feeddescription":
		case "feedimagetitle": // already used when handling "feedimageurl" attr
		case "feedimageurl":
			_, imgUrlFound := evt.Attributes["imageurl"]
			if !imgUrlFound { // use the feed image instead
				var imgTitle string
				attrImgTitle, imgTitleFound := evt.Attributes["feedimagetitle"]
				switch imgTitleFound {
				case true:
					imgTitle = attrImgTitle.GetCeString()
				default:
					imgTitle = "Image"
				}
				switch html {
				case true:
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
			}
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
			switch html {
			case true:
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
					txt += fmt.Sprintf("\n%s: true\n", attrName)
				default:
					txt += fmt.Sprintf("\n%s: false\n", attrName)
				}
			case *pb.CloudEventAttributeValue_CeInteger:
				txt += fmt.Sprintf("\n%s: %d\n", attrName, vt.CeInteger)
			case *pb.CloudEventAttributeValue_CeString:
				v := f.HtmlPolicy.Sanitize(vt.CeString)
				if trunc {
					v = truncateStringUtf8(v, fmtLenMaxAttrVal)
				}
				switch html {
				case true:
					txt += fmt.Sprintf("\n<b>%s</b>: %s\n", attrName, v)
				default:
					txt += fmt.Sprintf("\n%s: %s\n", attrName, v)
				}
			case *pb.CloudEventAttributeValue_CeUri:
				v := vt.CeUri
				if trunc {
					v = truncateStringUtf8(v, fmtLenMaxAttrVal)
				}
				txt += fmt.Sprintf("\n%s: %s\n", attrName, v)
			case *pb.CloudEventAttributeValue_CeUriRef:
				v := vt.CeUriRef
				if trunc {
					v = truncateStringUtf8(v, fmtLenMaxAttrVal)
				}
				txt += fmt.Sprintf("\n%s: %s\n", attrName, v)
			case *pb.CloudEventAttributeValue_CeTimestamp:
				v := vt.CeTimestamp
				txt += fmt.Sprintf("\n%s: %s\n", attrName, v.AsTime().Format(time.RFC3339))
			case *pb.CloudEventAttributeValue_CeBytes:
				v := base64.StdEncoding.EncodeToString(vt.CeBytes)
				if trunc {
					v = truncateStringUtf8(v, fmtLenMaxAttrVal)
				}
				txt += fmt.Sprintf("\n%s: %s\n", attrName, v)
			}
		}
	}

	txt += f.convertTextDataAndFooter(evt, html, trunc)

	return
}

func (f Format) convertTextDataAndFooter(evt *pb.CloudEvent, html bool, trunc bool) (txt string) {

	txtData := evt.GetTextData()
	switch {
	case txtData != "":
		txtData = f.HtmlPolicy.Sanitize(txtData)
		if trunc {
			txtData = truncateStringUtf8(txtData, fmtLenMaxBodyTxt)
		}
		txt += fmt.Sprintf("\n%s\n", txtData)
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
	if _, err := url.Parse(urlSrc); err == nil && html {
		txt += fmt.Sprintf("<a href=\"%s\">Source</a>\n", urlSrc)
	} else {
		txt += fmt.Sprintf("Source: %s\n", urlSrc)
	}

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
