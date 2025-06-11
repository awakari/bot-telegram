package messages

import (
	"fmt"
	"github.com/awakari/bot-telegram/model"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/microcosm-cc/bluemonday"
	"gopkg.in/telebot.v3"
	"net/url"
	"strings"
	"unicode/utf8"
)

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

const fmtLenMaxBodyTxt = 300
const tagCountMax = 8
const tagLenMax = 64

var htmlStripTags = bluemonday.
	StrictPolicy().
	AddSpaceWhenStrippingTag(true)

func (f Format) Convert(evt *pb.CloudEvent, subId, subDescr string, mode FormatMode) (tgMsg any) {
	fileTypeAttr, fileTypeFound := evt.Attributes[model.CeKeyTgFileType]
	if fileTypeFound && mode != FormatModeRaw {
		ft := FileType(fileTypeAttr.GetCeInteger())
		file := telebot.File{
			FileID:   evt.Attributes[model.CeKeyTgFileId].GetCeString(),
			UniqueID: evt.Attributes[model.CeKeyTgFileUniqueId].GetCeString(),
		}
		switch ft {
		case FileTypeAudio:
			tgMsg = &telebot.Audio{
				File:     file,
				Duration: int(evt.Attributes[model.CeKeyTgFileMediaDuration].GetCeInteger()),
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
				Width:   int(evt.Attributes[model.CeKeyTgFileImgWidth].GetCeInteger()),
				Height:  int(evt.Attributes[model.CeKeyTgFileImgHeight].GetCeInteger()),
				Caption: f.convert(evt, subId, subDescr, mode, false, false),
			}
		case FileTypeVideo:
			tgMsg = &telebot.Video{
				File:     file,
				Width:    int(evt.Attributes[model.CeKeyTgFileImgWidth].GetCeInteger()),
				Height:   int(evt.Attributes[model.CeKeyTgFileImgHeight].GetCeInteger()),
				Duration: int(evt.Attributes[model.CeKeyTgFileMediaDuration].GetCeInteger()),
				Caption:  f.convert(evt, subId, subDescr, mode, false, false),
			}
		}
	} else {
		_, msgFromTg := evt.Attributes[ceKeyTgMessageId]
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

func (f Format) convert(evt *pb.CloudEvent, interestId, descr string, mode FormatMode, trunc, attrs bool) (txt string) {

	if attrs {
		txt += f.header(evt, mode)
	}

	txtData := evt.GetTextData()
	if txtData != "" {
		switch mode {
		case FormatModeHtml:
			txtData = f.HtmlPolicy.Sanitize(txtData)
		default:
			txtData = htmlStripTags.Sanitize(txtData)
		}
		if txt != "" {
			txt += "\n\n"
		}
		txt += strings.TrimSpace(txtData)
	}
	attrName, attrNameFound := evt.Attributes["name"]
	if txt == "" && attrNameFound {
		txt = attrName.GetCeString()
	}

	if trunc {
		txt = truncateStringUtf8(txt, fmtLenMaxBodyTxt) + "\n"
	}

	attrCats, _ := evt.Attributes[model.CeKeyCategories]
	cats := strings.Split(attrCats.GetCeString(), " ")
	var tags []string
	var tagCount int
	for i, cat := range cats {
		if i == tagCountMax {
			break
		}
		var t string
		switch strings.HasPrefix(cat, "#") {
		case true:
			t = cat
		default:
			t = "#" + cat
		}
		if len(t) > 1 && len(t) < tagLenMax {
			tags = append(tags, t)
		}
		if tagCount > 10 {
			break
		}
		tagCount++
	}
	if len(tags) > 0 {
		txt += fmt.Sprintf("%s\n", strings.Join(tags, " "))
	}

	objAttr, objAttrFound := evt.Attributes["object"]
	var addrOrig string
	if objAttrFound {
		switch objAttr.Attr.(type) {
		case *pb.CloudEventAttributeValue_CeString:
			addrOrig = objAttr.GetCeString()
		case *pb.CloudEventAttributeValue_CeUri:
			addrOrig = objAttr.GetCeUri()
		}
	}
	if addrOrig == "" || (!strings.HasPrefix(addrOrig, "https://") && !strings.HasPrefix(addrOrig, "http://")) {
		objAttr, objAttrFound = evt.Attributes["objecturl"]
		if objAttrFound {
			switch objAttr.Attr.(type) {
			case *pb.CloudEventAttributeValue_CeString:
				addrOrig = objAttr.GetCeString()
			case *pb.CloudEventAttributeValue_CeUri:
				addrOrig = objAttr.GetCeUri()
			}
		}
	}
	if addrOrig == "" {
		addrOrig = evt.Source
	}
	if strings.Contains(evt.Type, "telegram") && strings.HasPrefix(addrOrig, "@") {
		addrOrig = "https://t.me/" + strings.TrimPrefix(addrOrig, "@")
	}
	addrMatch := f.UriReaderEvtBase + evt.Id + "&interestId=" + interestId
	addrInterest := "https://awakari.com/sub-details.html?id=" + interestId
	switch mode {
	case FormatModeHtml:
		txt += fmt.Sprintf(
			"\n<a href=\"%s\">Origin</a> | <a href=\"%s\">Interest</a> | <a href=\"%s\">Match</a>",
			addrOrig, addrInterest, addrMatch,
		)
	default:
		if len(addrOrig) > 100 {
			urlOrig, err := url.Parse(addrOrig)
			switch err {
			case nil:
				addrOrig = urlOrig.Scheme + urlOrig.Host
			default:
				addrOrig = addrOrig[0:100]
			}
		}
		txt += "\nOrigin: " + addrOrig
		txt += "\nInterest: " + addrInterest
		txt += "\nMatch: " + addrMatch
	}

	return
}

func (f Format) header(evt *pb.CloudEvent, mode FormatMode) (txt string) {

	attrHead, headPresent := evt.Attributes[model.CeKeyHeadline]
	if headPresent {
		txt = strings.TrimSpace(attrHead.GetCeString())
	}

	attrTitle, titlePresent := evt.Attributes[model.CeKeyTitle]
	if titlePresent {
		if txt != "" {
			txt += " "
		}
		txt += strings.TrimSpace(attrTitle.GetCeString())
	}

	if txt != "" && mode == FormatModeHtml {
		txt = fmt.Sprintf("<b>%s</b>", txt)
	}

	attrDescr, descrPresent := evt.Attributes[model.CeKeyDescription]
	if descrPresent {
		if txt != "" {
			txt += "\n\n"
		}
		txt += strings.TrimSpace(attrDescr.GetCeString())
	}

	attrSummary, attrSummaryFound := evt.Attributes[model.CeKeySummary]
	if attrSummaryFound {
		v := attrSummary.GetCeString()
		if v != attrTitle.GetCeString() && v != attrDescr.GetCeString() && v != evt.GetTextData() {
			switch mode {
			case FormatModeHtml:
				v = f.HtmlPolicy.Sanitize(v)
			default:
				v = htmlStripTags.Sanitize(v)
			}
			if txt != "" {
				txt += "\n\n"
			}
			txt += strings.TrimSpace(v)
		}
	}

	txt = f.HtmlPolicy.Sanitize(txt)

	return
}

func truncateStringUtf8(s string, lenMax int) string {
	s = strings.TrimSpace(s)
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
