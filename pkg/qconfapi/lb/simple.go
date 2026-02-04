package lb

import (
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/qiniu/xlog.v1"
)

type simple struct {
	*Config
	client *http.Client
	sel    *selector
}

var DefaultTransport = NewTransport(nil)

func newSimple(cfg *Config, tr http.RoundTripper) *simple {
	if cfg.FailRetryIntervalS == 0 {
		cfg.FailRetryIntervalS = DefaultFailRetryInterval
	}
	if cfg.ShouldRetry == nil {
		cfg.ShouldRetry = ShouldRetry
	}
	if cfg.MaxFails == 0 {
		cfg.MaxFails = 1
	}
	if cfg.MaxFailsPeriodS == 0 {
		cfg.MaxFailsPeriodS = 1
	}
	if tr == nil {
		tr = DefaultTransport
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(cfg.ClientTimeoutMS) * time.Millisecond,
	}
	return &simple{
		Config: cfg,
		client: client,
		sel:    newSelector(cfg.Hosts, cfg.TryTimes, cfg.FailRetryIntervalS, cfg.DnsResolve, cfg.DnsCacheTimeS, cfg.LookupHost, cfg.MaxFails, cfg.MaxFailsPeriodS),
	}
}

func (p *simple) doWithHostRet(req *Request) (rhost string, resp *http.Response, code int, err error) {

	ctx := req.Context()
	xl := xlog.FromContextSafe(ctx)

	reqURI := req.URL.RequestURI()
	httpreq := req.Request

	xl.Debug("simple.DoWithHostRet: start", reqURI)
	defer func() { xl.Debug("simple.DoWithHostRet: done", err) }()

	h, sel := p.sel.Get(xl)
	if h == nil {
		err = ErrServiceNotAvailable
		xl.Error("simple.DoWithHostRet: get host", err)
		return
	}
	rhost = h.raw

	tryTimes := p.sel.GetTryTimes()
	for i := uint32(0); i < tryTimes; i++ {

		httpreq.URL, err = url.Parse(rhost + reqURI)
		if err != nil {
			return
		}
		if req.Host == "" {
			if h.host != "" && p.DnsResolve && !p.LookupHostNotHoldHost {
				httpreq.Host = h.host
			} else {
				httpreq.Host = httpreq.URL.Host

				// rollback to raw host, such as c.host = "www.google.com"
				if httpreq.Host == "" {
					httpreq.Host = rhost
				}
			}
		}
		if req.Body != nil {
			r := &Reader{req.Body, 0}
			httpreq.Body = nopReadatCloser{r, r}
		} else {
			httpreq.Body = nil
		}
		xl.Debug("simple.DoWithHostRet: with host", rhost+reqURI, httpreq.Host)
		resp, err = p.client.Do(&httpreq)
		code = 0
		if resp != nil {
			code = resp.StatusCode
		}

		if isCancelled(xl, req) {
			xl.Info("request canceled, err: ", err)
			return
		}

		if p.ShouldRetry(code, err) {
			xl.Warn("simple.DoWithHostRet: retry host, times: ", i, "code: ", code, "err: ", err, "host:", httpreq.URL.String())

			h.SetFail(xl)
			h = sel.Get(xl)
			if h == nil {
				xl.Error("simple.DoWithHostRet: get retry host", ErrServiceNotAvailable)
			} else {
				rhost = h.raw
			}

			// 这时候不会再去重试，不能关闭 resp.Body
			if h == nil || i == tryTimes-1 {
				xl.Debug("simple.DoWithHostRet: no more try", h, i)
				return
			}
			if resp != nil {
				discardAndClose(resp.Body)
			}
			continue
		}

		if resp != nil {
			resp.Body = newBodyReader(xl, p.SpeedLimit, resp.Body, h)
		}
		return
	}
	return
}

type bodyReader struct {
	SpeedLimit
	Reader io.ReadCloser
	offset int64
	once   sync.Once
	tr     time.Duration
	h      *host
	xl     *xlog.Logger
}

func newBodyReader(xl *xlog.Logger, cfg SpeedLimit, rc io.ReadCloser, h *host) *bodyReader {
	r := new(bodyReader)
	r.SpeedLimit = cfg
	r.Reader = rc
	r.h = h
	if xl != nil {
		r.xl = xl
	} else {
		r.xl = xlog.NewDummy()
	}

	return r
}

func (r *bodyReader) Read(val []byte) (n int, err error) {
	start := time.Now()
	n, err = r.Reader.Read(val)
	r.tr += time.Now().Sub(start)
	r.offset += int64(n)
	if err != nil && r.offset > r.CalcSpeedSizeThresholdB {
		r.once.Do(func() {
			speed := float64(r.offset) / r.tr.Seconds()
			if speed < float64(r.BanHostBelowBps) {
				r.xl.Errorf("ban host(%s) below Bps: speed(%f) < banHostsBelowBps(%d)", r.h.raw, speed, r.BanHostBelowBps)
				r.h.SetFail(r.xl)
			}
		})
	}
	return
}

func (r *bodyReader) Close() error {
	return r.Reader.Close()
}

type nopReadatCloser struct {
	io.ReaderAt
	io.Reader
}

func (n nopReadatCloser) Close() error {
	return nil
}
