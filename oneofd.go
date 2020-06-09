package oneofd

import (
	"encoding/json"
	"github.com/go-resty/resty/v2"
	"log"
	"strconv"
	"time"
)

type oneofd struct {
	r        *resty.Client
	Login    string
	Password string
}

type KKT struct {
	Address  string `json:"address"`
	Kktregid string `json:"kktregid"`
}

type Receipt struct {
	ID       int
	KktRegId string
	FP       string
	FD       string
	Date     string
	Products []Product
	Link     string
	Price    int
	VatPrice int
}

type Product struct {
	Name       string
	Quantity   int
	Price      int
	Vat        int
	VatPrice   int
	TotalPrice int
	FP         string
	FD         string
	FN         string
	Time       string
}

type Auth struct {
	AuthToken string `json:"authToken"`
}

type Kkts struct {
	Id      int    `json:"id"`
	Title   string `json:"title"`
	Address string `json:"address"`
	KKMS    []Kkt  `json:"kkms"`
}

type Kkt struct {
	Id            int    `json:"id"`
	OrgOd         int    `json:"orgId"`
	RetailPlaceId int    `json:"retailPlaceId"`
	InternalName  string `json:"internalName"`
	OnlineStatus  int    `json:"onlineStatus"`
	Status        int    `json:"status"`
	FnsKkmId      string `json:"fnsKkmId"`
	BillingStatus int    `json:"billingStatus"`
}

type Receipts struct {
	Id string `json:"id"`
}

type ReceiptOfd struct {
	Ticket struct {
		TransactionDate      string            `json:"transactionDate"`
		FiscalDriveNumber    string            `json:"fiscalDriveNumber"`
		EcashTotalSum        float32           `json:"ecashTotalSum"`
		FP                   string            `json:"fiscalId"`
		FiscalDocumentNumber int               `json:"fiscalDocumentNumber"`
		TaxationType         int               `json:"taxationType"`
		NdsNo                float32           `json:"ndsNo"`
		Nds0                 float32           `json:"nds0"`
		Nds10                float32           `json:"nds10"`
		Nds18                float32           `json:"nds18"`
		Nds20                float32           `json:"nds20"`
		UserInn              string            `json:"userInn"`
		KktRegId             string            `json:"kktRegId"`
		CashTotalSum         float32           `json:"cashTotalSum"`
		TotalSum             float32           `json:"totalSum"`
		OperationType        int               `json:"operationType"`
		Items                []ProductDocument `json:"items"`
		QRCode               string            `json:"qrCode"`
	} `json:"ticket"`
}
type KktsOne struct {
	KKT   map[string][]kkt `json:"KKT"`
	Count int              `json:"count"`
}

type kkt struct {
	Address      string `json:"address"`
	Last         string `json:"last"`
	Kktregid     string `json:"kktregid"`
	Turnover     int    `json:"turnover"`
	ReceiptCount int    `json:"receiptCount"`
}

type Link struct {
	Link string `json:"link"`
}

type ProductDocument struct {
	Quantity    json.Number `json:"quantity"`
	Price       int         `json:"price"`
	NdsSum      int         `json:"ndsSum"`
	Name        string      `json:"name"`
	Sum         int         `json:"sum"`
	ProductType int         `json:"productType"`
	PaymentType int         `json:"paymentType"`
}

var globalAuth Auth

func OneOfd(login, password string) *oneofd {
	return &oneofd{
		r:        resty.New(),
		Login:    login,
		Password: password,
	}
}

func (ofd *oneofd) auth() {
	_, err := ofd.r.R().
		SetBody(map[string]interface{}{"login": ofd.Login, "password": ofd.Password}).
		SetResult(&globalAuth).
		Post("https://org.1-ofd.ru/api/user/login")

	/*for _, cookie := range resp.Cookies() {
		ofd.r.SetCookie(cookie)
	}*/
	if err != nil {
		log.Printf("1 ofd failed auth: %v", err)
	}
}

func (ofd *oneofd) GetReceipts(date time.Time) (receipts []Receipt, err error) {
	ofd.auth()
	kkts, err := ofd.getKKT(date)
	for _, kkt := range kkts {
		r, err := ofd.getDocuments(kkt.Kktregid, date)
		if err != nil {
			return receipts, err
		}
		receipts = append(receipts, r...)
	}
	return receipts, err
}

func (ofd *oneofd) getKKT(date time.Time) (kkt []KKT, err error) {
	k := []Kkts{}
	_, err = ofd.r.R().
		SetHeader("X-XSRF-TOKEN", globalAuth.AuthToken).
		SetResult(&k).
		Get("https://org.1-ofd.ru/api/retail-places/kkms")
	if err != nil {
		log.Printf("[OFDYA] GetKKT: %s", err.Error())
	}

	for _, value := range k {
		for _, v := range value.KKMS {
			kkt = append(kkt, KKT{
				Address:  value.Address,
				Kktregid: strconv.Itoa(v.Id),
			})
		}
	}

	return kkt, err
}

func startDay(t time.Time) int64 {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location()).UnixNano() / int64(time.Millisecond)
}

func endDay(t time.Time) int64 {
	year, month, day := t.Date()
	return time.Date(year, month, day, 23, 59, 59, 0, t.Location()).UnixNano() / int64(time.Millisecond)
}

func (ofd *oneofd) getDocuments(kkt string, date time.Time) (documents []Receipt, err error) {
	docs := []Receipts{}
	_, err = ofd.r.R().
		SetHeader("X-XSRF-TOKEN", globalAuth.AuthToken).
		SetResult(&docs).
		Get("https://org.1-ofd.ru/api/kkms/" + kkt + "/transactions?fromDate=" + strconv.FormatInt(startDay(date), 10) + "&toDate=" + strconv.FormatInt(endDay(date), 10))
	for _, v := range docs {
		doc := ofd.getReceipt(v.Id)
		documents = append(documents, doc)
	}
	return documents, err
}

func (ofd *oneofd) getReceipt(id string) (doc Receipt) {
	d := ReceiptOfd{}
	_, err := ofd.r.R().
		SetHeader("X-XSRF-TOKEN", globalAuth.AuthToken).
		SetResult(&d).
		Get("https://org.1-ofd.ru/api/ticket/" + id)
	if err != nil {
		log.Printf("[1OFD] getReceipt: %s", err.Error())
	}

	date, err := time.Parse("2006-01-02T15:04:05.000", d.Ticket.TransactionDate)
	if err != nil {
		log.Printf("[1OFD] getReceipt->parseDate:(%s) %s", d.Ticket.TransactionDate, err.Error())
	}
	doc.Date = date.Format(time.RFC3339)
	doc.KktRegId = d.Ticket.KktRegId
	doc.FD = strconv.Itoa(d.Ticket.FiscalDocumentNumber)
	doc.FP = d.Ticket.FP
	if d.Ticket.Nds20 > 0 {
		doc.VatPrice = int(d.Ticket.Nds20) * 100
	} else if d.Ticket.Nds0 > 0 {
		doc.VatPrice = int(d.Ticket.Nds0) * 100
	} else if d.Ticket.Nds10 > 0 {
		doc.VatPrice = int(d.Ticket.Nds10) * 100
	}
	doc.Price = int(d.Ticket.TotalSum) * 100
	for _, v := range d.Ticket.Items {
		q, _ := v.Quantity.Float64()
		doc.Products = append(doc.Products, Product{
			Name:       v.Name,
			Quantity:   int(q),
			Price:      v.Price,
			Vat:        0,
			VatPrice:   v.NdsSum,
			TotalPrice: doc.Price,
			FP:         doc.FP,
			FD:         doc.FD,
			Time:       date.Format(time.RFC3339),
		})
	}

	doc.Link = "https://consumer.1-ofd.ru/v1?" + d.Ticket.QRCode

	return doc
}
