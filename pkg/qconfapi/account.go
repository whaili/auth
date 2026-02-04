package qconfapi

// import (
// 	"net/http"
// 	"sync"
// 	"time"

// 	"chat/qconfapi/oauth"
// )

// // Transport custom transport implement RoundTripper interface which request
// // with acc token, for simplistic, just exchange token use username and password
// type Transport struct {
// 	mu sync.Mutex
// 	tr *oauth.Transport
// }

// // AccConfig account service config for exchange token
// type AccConfig struct {
// 	Host         string `yaml:"host" json:"host"`
// 	UserName     string `yaml:"username" json:"username"`
// 	Password     string `yaml:"password" json:"password"`
// 	ClientID     string `yaml:"client_id" json:"client_id"`
// 	ClientSecret string `yaml:"client_secret" json:"client_secret"`
// 	// AutoRefreshInterval token 自动刷新时间间隔
// 	// 可选参数：若设置为 0，则不自动刷新，若设置为 x（x>0），则会以 max {x, 50*time.Minute} 的间隔自动刷新
// 	AutoRefreshInterval time.Duration `yaml:"auto_refresh_interval" json:"auto_refresh_interval"`
// }

// // autoRefreshToken 间隔 interval 自动刷新 token
// func (t *Transport) autoRefreshToken(interval time.Duration) {
// 	if interval < 50*time.Minute {
// 		interval = 50 * time.Minute
// 	}

// 	go func() {
// 		ticker := time.NewTicker(interval)
// 		defer ticker.Stop()

// 		for {
// 			select {
// 			case <-ticker.C:
// 				_ = t.refreshToken(t.tr.RefreshToken)
// 			}
// 		}
// 	}()
// }

// func (t *Transport) refreshToken(refreshToken string) error {
// 	if t.tokenExpired() {
// 		t.mu.Lock()
// 		if t.tokenExpired() {
// 			_, _, err := t.tr.ExchangeByRefreshToken(refreshToken)
// 			if err != nil {
// 				t.mu.Unlock()
// 				return err
// 			}
// 		}
// 		t.mu.Unlock()
// 	}
// 	return nil
// }

// // RoundTrip refresh token if token has expired
// func (t *Transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
// 	if err = t.refreshToken(t.tr.RefreshToken); err != nil {
// 		return nil, err
// 	}
// 	return t.tr.RoundTrip(req)
// }

// // tokenExpired
// func (t *Transport) tokenExpired() bool {
// 	return t.tr.TokenExpiry > 0 && t.tr.TokenExpiry-120 <= qtime.Seconds()
// }

// // NewTransport with AccConfig
// func NewTransport(conf *AccConfig) (*Transport, error) {
// 	return NewWithTransport(conf, oauth.DefaultTransport)
// }

// // NewWithTransport creates transport with AccConfig and initial transport
// func NewWithTransport(conf *AccConfig, tr http.RoundTripper) (*Transport, error) {
// 	cfg := &oauth.Config{
// 		TokenURL: conf.Host + "/oauth2/token",
// 	}

// 	transport := &oauth.Transport{Config: cfg}
// 	if tr != nil {
// 		transport.Transport = tr
// 	}
// 	_, _, err := transport.ExchangeByPassword(conf.UserName, conf.Password)
// 	if err != nil {
// 		return nil, err
// 	}

// 	t := &Transport{tr: transport}

// 	if conf.AutoRefreshInterval > 0 {
// 		t.autoRefreshToken(conf.AutoRefreshInterval)
// 	}

// 	return t, err
// }
