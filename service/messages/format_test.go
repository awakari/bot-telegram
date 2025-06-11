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
			out: `⚠ Daily publishing limit reached.

Increase your daily publication limit or nominate own sources for the dedicated limit.

If you did not publish messages, <a href="https://awakari.com/pub.html?own=true" rel="nofollow">check own publication sources</a> you added.

<a href="https://bbs.archlinux.org/extern.php?action=feed&fid=32&type=atom">Origin</a> | <a href="https://awakari.com/sub-details.html?id=sub1">Interest</a> | <a href="82f39262-5eb4-4f7f-9142-7c489d670907&interestId=sub1">Match</a>`,
		},
		"2": {
			in: &pb.CloudEvent{
				SpecVersion: "1.0",
				Id:          "QPpQBtMTmjL6xaJyA5OpQcPo1qa",
				Source:      "https://www.nanowerk.com/nwfeedcomplete.xml",
				Type:        "com_awakari_feeds_v1",
				Attributes: map[string]*pb.CloudEventAttributeValue{
					"objecturl": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "https://www.nanowerk.com/news2/space/newsid=67030.php",
						},
					},
					"snippet": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "\nNanowerk Nanotechnology and Emerging Technologies News\nSilicate clouds discovered in atmosphere of distant exoplanet\nhttps://www.nanowerk.com/news2/space/newsid=67030.php\nAstrophysicists have gained precious new insights into how distant exoplanets form and what their atmospheres can look like, after using the James Webb Telescope to image two young exoplanets in extraordinary detail.\nNanowerk Nanotechnology and Emerging Technologies News\nNanowerk Nanotechnology and Emerging Technologies news headlines from Nanowerk",
						},
					},
					"sourcedescription": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "Nanowerk Nanotechnology and Emerging Technologies news headlines from Nanowerk",
						},
					},
					"sourceimagetitle": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "Nanowerk Nanotechnology and Emerging Technologies News",
						},
					},
					"sourceimageurl": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "https://www.nanowerk.com/images/NWlogo.jpg",
						},
					},
					"sourcetitle": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "Nanowerk Nanotechnology and Emerging Technologies News",
						},
					},
					"srccopyright": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "Copyright Nanowerk LLC",
						},
					},
					"summary": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "Astrophysicists have gained precious new insights into how distant exoplanets form and what their atmospheres can look like, after using the James Webb Telescope to image two young exoplanets in extraordinary detail.",
						},
					},
					"title": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "Silicate clouds discovered in atmosphere of distant exoplanet",
						},
					},
				},
			},
			out: "<b>Silicate clouds discovered in atmosphere of distant exoplanet</b>\n\nAstrophysicists have gained precious new insights into how distant exoplanets form and what their atmospheres can look like, after using the James Webb Telescope to image two young exoplanets in extraordinary detail.\n\n<a href=\"https://www.nanowerk.com/news2/space/newsid=67030.php\">Origin</a> | <a href=\"https://awakari.com/sub-details.html?id=sub1\">Interest</a> | <a href=\"QPpQBtMTmjL6xaJyA5OpQcPo1qa&interestId=sub1\">Match</a>",
		},
		"3": {
			in: &pb.CloudEvent{
				SpecVersion: "1.0",
				Id:          "2yMTtfDHfZHnpTEdSVe8J90Qc6r",
				Source:      "@rabota_razrabotchika",
				Type:        "com_awakari_source_telegram_v1_0",
				Attributes: map[string]*pb.CloudEventAttributeValue{
					"data": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "Golang Developer (Middle)  \nЗаработная плата по договоренности  \n\nОбязанности:  \n• Опыт работы на аналогичной позиции, а также:  \n• Опыт разработки Golang от 2 лет – Middle.  \n\nТребования:  \n• Уверенные знания gRPC, Gorm, Gin;  \n• Опыт работы с Postgres, Redis, RabbitMQ, Kafka, ClickHouse;  \n• Умение работать в команде.  \n\nМы предлагаем:  \n• Официальное оформление с первого рабочего дня (испытательный срок 3 месяца);  \n• Крутую команду с компетентными, креативными и веселыми коллегами;  \n• Достойную белую заработную плату;  \n• Трудовой отпуск 28 календарных дней и 3 sick days в году;  \n• Комфортабельный офис в самом центре Минска;  \n• Возможность прохождения обучения за счет компании;  \n• Добровольное медицинское страхование (после прохождения испытательного срока);  \n• Материальная компенсация занятий спортом (до 150 бел. рублей в месяц) и культурно-массовых мероприятий (до 150 бел. рублей в месяц) после прохождения испытательного срока;  \n• Регулярные корпоративные праздники, мероприятия и приятные подарки от компании.  \n\nКонтакты:  \n+375295373920  \ndarya.buka@indev.by",
						},
					},
					"language": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "ru",
						},
					},
					"sentiment": {
						Attr: &pb.CloudEventAttributeValue_CeInteger{
							CeInteger: -51,
						},
					},
					"snippet": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "Golang Developer (Middle) Заработная плата по договоренности Обязанности: • Опыт работы на аналогичной позиции, а также: • Опыт разработки Golang от 2 лет – Middle. Требования: • Уверенные знания gRPC, Gorm, Gin; • Опыт работы с Postgres, Redis, RabbitMQ, Kafka, ClickHouse; • Умение работать в команде. Мы предлагаем: • Официальное оформление с первого рабочего дня (испытательный срок 3 месяца); • Крутую команду с компетентными, креативными и веселыми коллегами; • Достойную белую заработную плату; • Трудовой отпуск 28 календарных дней и 3 sick days в году; • Комфортабельный офис в самом центре Минска; • В",
						},
					},
					"tgmessageid": {
						Attr: &pb.CloudEventAttributeValue_CeString{
							CeString: "2325741568",
						},
					},
				},
			},
			out: `
<a href="https://t.me/rabota_razrabotchika">Origin</a> | <a href="https://awakari.com/sub-details.html?id=sub1">Interest</a> | <a href="2yMTtfDHfZHnpTEdSVe8J90Qc6r&interestId=sub1">Match</a>`,
		},
	}
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			out := fmtMsg.Convert(c.in, "sub1", "sub1 description", FormatModeHtml)
			assert.Equal(t, c.out, out)
		})
	}
}
