package geolbot

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Unknwon/com"
	"github.com/zhuharev/jober/jobs"
	"github.com/zhuharev/jober/types"

	ini "gopkg.in/ini.v1"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

func init() {
	jobs.Register("geolbot", New())
}

type Robot struct {
	token   string
	bot     *tgbotapi.BotAPI
	adminID int64
}

func New() *Robot {
	return &Robot{}
}

func (r *Robot) Init(cfg *ini.Section) (types.Job, error) {
	time.Sleep(1 * time.Second)
	r.token = cfg.Key("token").String()
	r.adminID = cfg.Key("admin_id").MustInt64()

	var err error
	r.bot, err = tgbotapi.NewBotAPIWithClient(r.token, &http.Client{})
	if err != nil {

		log.Println(err)
		return r, nil
	}

	msg := tgbotapi.NewMessage(r.adminID, "Geo bot started")
	_, err = r.bot.Send(msg)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Robot) Run() (err error) {

	log.Println("disable debug")
	r.bot.Debug = false

	log.Printf("Authorized on account %s\n", r.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := r.bot.GetUpdatesChan(u)
	if err != nil {
		return err
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}
		err = r.handler(update.Message)
		if err != nil {
			log.Println(err)
		}
	}
	return nil
}

func (r *Robot) handler(m *tgbotapi.Message) error {

	log.Println(m.Text)
	var uFmt = `https://geocode-maps.yandex.ru/1.x/?geocode=%s&lang=ru_RU&format=json&results=1&ll=30.310623,59.940007`

	url := fmt.Sprintf(uFmt, m.Text)
	var res Res
	err := com.HttpGetJSON(http.DefaultClient, url, &res)
	if err != nil {
		panic(err)
	}
	pos := res.Response.GeoObjectCollection.FeatureMember[0].GeoObject.Point.Pos
	arr := strings.Split(pos, " ")

	if len(arr) != 2 {
		return nil
	}

	f1, _ := strconv.ParseFloat(arr[1], 64)
	f2, _ := strconv.ParseFloat(arr[0], 64)
	//addr := "https://static-maps.yandex.ru/1.x/?ll=" + arr[0] + "," + arr[1] + "&spn=0.05,0.05&l=map&size=300,450"
	msg := tgbotapi.NewLocation(m.Chat.ID, f1, f2)
	//msg := tgbotapi.NewMessage(m.Chat.ID, addr)
	_, err = r.bot.Send(msg)
	return err
}

type Res struct {
	Response struct {
		GeoObjectCollection struct {
			MetaDataProperty struct {
				GeocoderResponseMetaData struct {
					Request   string `json:"request"`
					Found     string `json:"found"`
					Results   string `json:"results"`
					BoundedBy struct {
						Envelope struct {
							LowerCorner string `json:"lowerCorner"`
							UpperCorner string `json:"upperCorner"`
						} `json:"Envelope"`
					} `json:"boundedBy"`
				} `json:"GeocoderResponseMetaData"`
			} `json:"metaDataProperty"`
			FeatureMember []struct {
				GeoObject struct {
					MetaDataProperty struct {
						GeocoderMetaData struct {
							Kind      string `json:"kind"`
							Text      string `json:"text"`
							Precision string `json:"precision"`
							Address   struct {
								CountryCode string `json:"country_code"`
								Formatted   string `json:"formatted"`
								Components  []struct {
									Kind string `json:"kind"`
									Name string `json:"name"`
								} `json:"Components"`
							} `json:"Address"`
							AddressDetails struct {
								Country struct {
									AddressLine        string `json:"AddressLine"`
									CountryNameCode    string `json:"CountryNameCode"`
									CountryName        string `json:"CountryName"`
									AdministrativeArea struct {
										AdministrativeAreaName string `json:"AdministrativeAreaName"`
										Locality               struct {
											LocalityName string `json:"LocalityName"`
											Thoroughfare struct {
												ThoroughfareName string `json:"ThoroughfareName"`
											} `json:"Thoroughfare"`
										} `json:"Locality"`
									} `json:"AdministrativeArea"`
								} `json:"Country"`
							} `json:"AddressDetails"`
						} `json:"GeocoderMetaData"`
					} `json:"metaDataProperty"`
					Description string `json:"description"`
					Name        string `json:"name"`
					BoundedBy   struct {
						Envelope struct {
							LowerCorner string `json:"lowerCorner"`
							UpperCorner string `json:"upperCorner"`
						} `json:"Envelope"`
					} `json:"boundedBy"`
					Point struct {
						Pos string `json:"pos"`
					} `json:"Point"`
				} `json:"GeoObject"`
			} `json:"featureMember"`
		} `json:"GeoObjectCollection"`
	} `json:"response"`
}
