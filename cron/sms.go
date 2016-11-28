package cron

import (
	"github.com/nxintech/sender/g"
	"github.com/nxintech/sender/model"
	"github.com/nxintech/sender/proc"
	"github.com/nxintech/sender/redis"
	"github.com/astaxie/beego/httplib"
	"log"
	"time"
	"strconv"
	"encoding/hex"
	"encoding/xml"
	"crypto/md5"
)

type ShortMessage struct {
	SendSort             string `xml:"sendSort"`
	SendType             string `xml:"sendType"`
	IsSwitchChannelRetry int    `xml:"isSwitchChannelRetry"`
	IsGroup              int    `xml:"isGroup"`
	PhoneNumber          string `xml:"phoneNumber"`
	Message              string `xml:"message"`
	Remarks              string `xml:"remarks"`
}

func ConsumeSms() {
	queue := g.Config().Queue.Sms
	for {
		L := redis.PopAllSms(queue)
		if len(L) == 0 {
			time.Sleep(time.Millisecond * 200)
			continue
		}
		SendSmsList(L)
	}
}

func SendSmsList(L []*model.Sms) {
	for _, sms := range L {
		SmsWorkerChan <- 1
		go SendSms(sms)
	}
}

func SendSms(sms *model.Sms) {
	defer func() {
		<-SmsWorkerChan
	}()

	url := g.Config().Api.Sms
	sysId := g.Config().Sms.SysId
	secret := g.Config().Sms.Secret
	debug := g.Config().Sms.Debug

	// timestamp
	timeStamp := time.Now().UnixNano() / int64(time.Millisecond)
	timeStampStr := strconv.FormatInt(timeStamp, 10)

	// token
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(secret + timeStampStr))
	cipherStr := md5Ctx.Sum(nil)
	token := hex.EncodeToString(cipherStr)

	// xml 构造
	message := &ShortMessage{
		SendSort: "SMS",
		SendType: "COMMON_GROUP",
		IsSwitchChannelRetry: 1,
		IsGroup: 1,
		PhoneNumber: sms.Tos,
		Message: sms.Content,
		Remarks: sysId,
	}
	xmlOutPut, _ := xml.Marshal(message)

	r := httplib.Post(url).SetTimeout(5*time.Second, 2*time.Minute).Debug(debug)
	r.Param("message", string(xmlOutPut))
	r.Param("timestamp", timeStampStr)
	r.Param("systemId", sysId)
	r.Param("accessToken", token)
	r.Param("businessChannel", "OTHERS")
	resp, err := r.String()
	if err != nil {
		log.Println(err)
	}

	proc.IncreSmsCount()

	if g.Config().Debug {
		log.Println("==sms==>>>>", sms)
		log.Println("<<<<==sms==", resp)
		log.Println(r.DumpRequest())
	}

}
