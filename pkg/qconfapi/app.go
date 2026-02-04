package qconfapi

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/qiniu/xlog.v1"
	"qiniu.com/auth/proto.v1"
)

type AppInfo struct {
	Uid     uint32 `json:"uid"     bson:"uid"`
	AppName string `json:"appName" bson:"appName"`
	AppId   uint32 `json:"appId"   bson:"appId"`

	Key          string    `json:"key,omitempty"           bson:"key,omitempty"`
	Secret       string    `json:"secret,omitempty"        bson:"secret,omitempty"`
	State        uint16    `json:"state,omitempty"         bson:"state,omitempty"`
	LastModified time.Time `json:"last-modified,omitempty" bson:"last-modified,omitempty"`
	CreationTime time.Time `json:"creation-time,omitempty" bson:"creation-time,omitempty"`
	Comment      string    `json:"comment,omitempty"       bson:"comment,omitempty"`

	Key2          string    `json:"key2,omitempty"           bson:"key2,omitempty"`
	Secret2       string    `json:"secret2,omitempty"        bson:"secret2,omitempty"`
	State2        uint16    `json:"state2,omitempty"         bson:"state2,omitempty"`
	LastModified2 time.Time `json:"last-modified2,omitempty" bson:"last-modified2,omitempty"`
	CreationTime2 time.Time `json:"creation-time2,omitempty" bson:"creation-time2,omitempty"`
	Comment2      string    `json:"comment2,omitempty"       bson:"comment2,omitempty"`
}

type AccessInfo struct {
	Key    string `json:"key"`
	Secret string `json:"secret"`
}

func (p *Client) GetAppInfo(l *xlog.Logger, app string, uid uint32) (ret AppInfo, err error) {
	err = p.Get(l, &ret, MakeId(app, uid), 0)
	return
}

func (p *Client) GetAkSk(l *xlog.Logger, uid uint32) (ak, sk string, err error) {

	info, err := p.GetAppInfo(l, "default", uid)
	if err != nil {
		return
	}

	ak = info.Key
	sk = info.Secret
	return
}

// ------------------------------------------------------------------------

const groupPrefix = "app:"
const groupPrefixLen = len(groupPrefix)

var (
	ErrInvalidGroup = errors.New("invalid group")
	ErrInvalidId    = errors.New("invalid id")
)

func MakeId(app string, uid uint32) string {
	return groupPrefix + strconv.FormatUint(uint64(uid), 36) + ":" + app
}

// ------------------------------------------------------------------------

type DisabledType int
type Second int64

const (
	DISABLED_TYPE_AUTO           DisabledType = 0 // 冻结后允许充值自动解冻
	DISABLED_TYPE_MANUAL         DisabledType = 1 // 冻结后需要手动解冻
	DISABLED_TYPE_PARENT         DisabledType = 2 // 被父账号冻结
	DISABLED_TYPE_OVERDUE        DisabledType = 3
	DISABLED_TYPE_NONSTD_OVERDUE DisabledType = 4
)

func (t DisabledType) Humanize() string {
	switch t {
	case DISABLED_TYPE_AUTO:
		return "欠费冻结"
	case DISABLED_TYPE_MANUAL:
		return "非欠费冻结"
	case DISABLED_TYPE_PARENT:
		return "被父账号冻结"
	case DISABLED_TYPE_OVERDUE:
		return "实时计费远超余额"
	case DISABLED_TYPE_NONSTD_OVERDUE:
		return "未认证用户超过免费额度"
	default:
		return fmt.Sprintf("unknown DisabledType: %d", t)
	}
}

type AccountInfo struct {
	Id               string       `bson:"id"                 json:"id"`              // 用户名(UserName)。唯一。
	Email            string       `bson:"email"              json:"email"`           // 电子邮箱。唯一。
	Username         string       `bson:"username"           json:"username"`        // 用户名。唯一。
	CreatedAt        Second       `bson:"ctime"              json:"ctime"`           // 用户创建时间。
	UpdatedAt        Second       `bson:"etime"              json:"etime"`           // 最后一次修改时间。
	LastLoginAt      Second       `bson:"lgtime"             json:"lgtime"`          // 最后一次登录时间。
	Uid              uint32       `bson:"uid"                json:"uid"`             // 用户数字ID。唯一。
	Utype            uint32       `bson:"utype"              json:"utype"`           // 用户类型。
	ParentUid        uint32       `bson:"parent_uid"         json:"parent_uid"`      // 父用户Uid
	Activated        bool         `bson:"activated"          json:"activated"`       // 用户是否已经激活。
	DisabledType     DisabledType `bson:"disabled_type"      json:"disabled_type"`   // 用户冻结类型
	DisabledReason   string       `bson:"disabled_reason"    json:"disabled_reason"` // 用户冻结原因
	DisabledAt       time.Time    `bson:"disabled_at"        json:"disabled_at"`     // 用户冻结时间
	Vendors          []Vendor     `bson:"vendors"            json:"vendors"`
	ChildEmailDomain string       `bson:"child_email_domain" json:"child_email_domain"`
	CanGetChildKey   bool         `bson:"can_get_child_key"  json:"can_get_child_key"`
}

type Vendor struct {
	Vendor      string    `bson:"vendor"       json:"vendor"`
	VendorId    string    `bson:"vendor_id"    json:"vendor_id"`
	VendorEmail string    `bson:"vendor_email" json:"vendor_email"`
	CreatedAt   time.Time `bson:"created_at"   json:"created_at"`
}

const (
	// user type
	USER_TYPE_QBOX         = 0
	USER_TYPE_ADMIN        = proto.USER_TYPE_ADMIN
	USER_TYPE_VIP          = proto.USER_TYPE_VIP
	USER_TYPE_STDUSER      = proto.USER_TYPE_STDUSER
	USER_TYPE_STDUSER2     = proto.USER_TYPE_STDUSER2
	USER_TYPE_EXPUSER      = proto.USER_TYPE_EXPUSER
	USER_TYPE_PARENTUSER   = proto.USER_TYPE_PARENTUSER
	USER_TYPE_OP           = proto.USER_TYPE_OP
	USER_TYPE_SUPPORT      = proto.USER_TYPE_SUPPORT
	USER_TYPE_CC           = proto.USER_TYPE_CC
	USER_TYPE_QCOS         = proto.USER_TYPE_QCOS
	USER_TYPE_PILI         = proto.USER_TYPE_PILI
	USER_TYPE_FUSION       = proto.USER_TYPE_FUSION
	USER_TYPE_PANDORA      = proto.USER_TYPE_PANDORA
	USER_TYPE_DISTRIBUTION = proto.USER_TYPE_DISTRIBUTION
	USER_TYPE_QVM          = proto.USER_TYPE_QVM
	USER_TYPE_DISABLED     = proto.USER_TYPE_DISABLED
	USER_TYPE_OVERSEAS     = proto.USER_TYPE_OVERSEAS
	USER_TYPE_OVERSEAS_STD = proto.USER_TYPE_OVERSEAS_STD

	USER_TYPE_USERS            = proto.USER_TYPE_USERS
	USER_TYPE_SUDOERS          = proto.USER_TYPE_SUDOERS
	USER_TYPE_ENTERPRISE       = USER_TYPE_STDUSER
	USER_TYPE_ENTERPRISE_VUSER = USER_TYPE_STDUSER2

	USER_TYPE_UNREGISTERED = proto.USER_TYPE_UNREGISTERED
	USER_TYPE_BUFFERED     = proto.USER_TYPE_BUFFERED
)

func (i *AccountInfo) IsDisabled() bool {
	return i.Utype&USER_TYPE_DISABLED != 0
}

func (i *AccountInfo) Disable() {
	i.Utype |= USER_TYPE_DISABLED
}

func (i *AccountInfo) Enable() {
	i.Utype &^= USER_TYPE_DISABLED
}

func (i *AccountInfo) IsBuffered() bool {
	return i.Utype&USER_TYPE_BUFFERED > 0
}

func (i *AccountInfo) IsStdUser() bool {
	return i.GetCustomerGroup() == CUSTOMER_GROUP_NORMAL
}

func (i *AccountInfo) GetCustomerGroup() CustomerGroup {
	if i.Utype&USER_TYPE_USERS == 0 {
		return CUSTOMER_GROUP_INVALID
	}
	if i.Utype&USER_TYPE_EXPUSER != 0 {
		return CUSTOMER_GROUP_EXP
	}
	if i.Utype&USER_TYPE_VIP != 0 {
		return CUSTOMER_GROUP_VIP
	}
	return CUSTOMER_GROUP_NORMAL
}

func (p *Client) GetAccountInfo(l *xlog.Logger, uid uint32) (ret AccountInfo, err error) {
	err = p.Get(l, &ret, MakeUId(uid), 0)
	return
}

type UserType uint32

func (t UserType) IsDisabled() bool {
	return t&USER_TYPE_DISABLED != 0
}

// 无效用户
func (t UserType) IsInvalid() bool {
	return getCustomerGroup(t) == CUSTOMER_GROUP_INVALID
}

func (t UserType) IsBuffered() bool {
	return t&USER_TYPE_BUFFERED > 0
}

func (t UserType) IsOverseas() bool {
	return t&USER_TYPE_OVERSEAS > 0
}

func (t UserType) IsOverseasStd() bool {
	return t&USER_TYPE_OVERSEAS_STD > 0
}

func (t UserType) IsUnregistered() bool {
	return t&USER_TYPE_UNREGISTERED > 0
}

// ------------------------------------------------------------------------

const accGroupPrefix = "acc:"
const accGroupPrefixLen = len(groupPrefix)

func MakeUId(uid uint32) string {
	return accGroupPrefix + strconv.FormatUint(uint64(uid), 36)
}

// ------------------------------------------------------------------------

type AccountAccessInfo struct {
	Secret []byte `bson:"secret"`
	Appid  uint64 `bson:"appId,omitempty"`
	Uid    uint32 `bson:"uid"`
}

func (p *Client) GetAccessInfo(l *xlog.Logger, accessKey string) (ret AccountAccessInfo, err error) {

	err = p.Get(l, &ret, MakeAKId(accessKey), Cache_NoSuchEntry)
	return
}

// ------------------------------------------------------------------------

const akGroupPrefix = "ak:"
const akGroupPrefixLen = len(groupPrefix)

//var ErrInvalidGroup = errors.New("invalid group")

func MakeAKId(key string) string {
	return akGroupPrefix + key
}

func ParseId(id string) (key string, err error) {
	if !strings.HasPrefix(id, groupPrefix) {
		return "", ErrInvalidGroup
	}
	return id[groupPrefixLen:], nil
}

// ------------------------------------------------------------------------
