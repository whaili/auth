package sufymac

import (
	"fmt"
	"net/http"

	"github.com/qbox/mikud-live/common/sufy/sufysigner"
)

var ACCESS_KEY string
var SECRET_KEY string

// ---------------------------------------------------------------------------------------

type Mac struct {
	AccessKey string
	SecretKey string
}

type Transport struct {
	mac       Mac
	Transport http.RoundTripper
}

func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {

	_, err = sufysigner.Sign(req, t.mac.SecretKey, sufysigner.SignOption().SetAuthorizationHeader(true, t.mac.AccessKey))
	if err != nil {
		return nil, fmt.Errorf("req sign fail:%s %s", req.URL, err.Error())
	}
	//fmt.Println("sufy token:", req.Header.Get("Authorization"))
	return t.Transport.RoundTrip(req)
}

func (t *Transport) NestedObject() interface{} {

	return t.Transport
}

func NewTransport(mac *Mac, transport http.RoundTripper) *Transport {

	if transport == nil {
		transport = http.DefaultTransport
	}
	t := &Transport{Transport: transport}
	if mac == nil {
		t.mac.AccessKey = ACCESS_KEY
		t.mac.SecretKey = SECRET_KEY
	} else {
		t.mac = *mac
	}
	return t
}

func NewClient(mac *Mac, transport http.RoundTripper) *http.Client {

	t := NewTransport(mac, transport)
	return &http.Client{Transport: t}
}
