package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/qbox/mikud-live/common/dal"
	"github.com/qiniu/qmgo"
	uuid "github.com/satori/go.uuid"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	maxUidKeyCnt = 100
)

var (
	ErrApikeyNotFound       = errors.New("apikey not found")
	ErrExceedMaxUidKeyCount = errors.New("exceed max uid key count")
	ErrNameTooLong          = errors.New("name too long")
)

type ApikeySrv struct {
	qmgo       *qmgo.Client
	apikeyColl *qmgo.Collection
}

func NewApikeySrv(cfg *dal.MongoCfg) *ApikeySrv {
	qmgo, err := dal.NewMongoClient(cfg)
	if err != nil {
		panic(err)
	}

	apikeyColl := qmgo.Database(cfg.DB).Collection(cfg.Coll)

	apikeySrv := &ApikeySrv{
		qmgo:       qmgo,
		apikeyColl: apikeyColl,
	}
	return apikeySrv
}

type Apikey struct {
	Id        string    `json:"id" bson:"_id"`
	Key       string    `json:"key" bson:"key"`
	Name      string    `json:"name" bson:"name"`
	Uid       uint32    `json:"uid" bson:"uid"`
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
}

func (s *ApikeySrv) GetKey(ctx context.Context, key string) (*Apikey, error) {
	var apikeyInfo Apikey
	err := s.apikeyColl.Find(ctx, bson.M{"key": key}).One(&apikeyInfo)
	if err != nil {
		if errors.Is(err, qmgo.ErrNoSuchDocuments) {
			return nil, ErrApikeyNotFound
		}
		return nil, err
	}
	return &apikeyInfo, nil
}

func (s *ApikeySrv) ListApikeysByUid(ctx context.Context, uid uint32) ([]*Apikey, error) {
	var apikeys []*Apikey
	err := s.apikeyColl.Find(ctx, bson.M{"uid": uid}).All(&apikeys)
	if err != nil {
		return nil, err
	}
	return apikeys, nil
}

// 获取一个 uid 下有多少key
func (s *ApikeySrv) getUidKeyCnt(ctx context.Context, uid uint32) (int64, error) {
	return s.apikeyColl.Find(ctx, bson.M{"uid": uid}).Count()
}

func (s *ApikeySrv) CreateApikey(ctx context.Context, uid uint32, name string) (*Apikey, error) {
	uidCnt, err := s.getUidKeyCnt(ctx, uid)
	if err != nil {
		return nil, err
	}

	if uidCnt > maxUidKeyCnt {
		return nil, ErrExceedMaxUidKeyCount
	}

	if len(name) < 1 || len(name) > 20 {
		return nil, ErrNameTooLong
	}

	var apikey Apikey
	apikey.Id = uuid.NewV4().String()
	apikey.Key = generateApiKey(uid)
	apikey.Uid = uid
	apikey.Name = name
	apikey.CreatedAt = time.Now().Local()
	_, err = s.apikeyColl.InsertOne(ctx, apikey)
	if err != nil {
		return nil, err
	}
	return &apikey, nil
}

func (s *ApikeySrv) DeleteApikey(ctx context.Context, uid uint32, id string) error {
	var apikeyInfo Apikey
	err := s.apikeyColl.Find(ctx, bson.M{"uid": uid, "_id": id}).One(&apikeyInfo)
	if err != nil {
		if errors.Is(err, qmgo.ErrNoSuchDocuments) {
			return ErrApikeyNotFound
		}
		return err
	}

	err = s.apikeyColl.Remove(ctx, bson.M{"uid": uid, "_id": id})
	if err != nil {
		return err
	}
	return nil
}

func (s *ApikeySrv) RenameApikey(ctx context.Context, uid uint32, id string, name string) error {
	var apikeyInfo Apikey
	err := s.apikeyColl.Find(ctx, bson.M{"uid": uid, "_id": id}).One(&apikeyInfo)
	if err != nil {
		if errors.Is(err, qmgo.ErrNoSuchDocuments) {
			return ErrApikeyNotFound
		}
		return err
	}
	err = s.apikeyColl.UpdateOne(ctx, bson.M{"uid": uid, "_id": id}, bson.M{"$set": bson.M{"name": name}})
	if err != nil {
		return err
	}
	return nil
}

func intnSeed(n int, seed int) int {
	// unix nano is the same if called in the same nanosecond, so we need to add another random seed
	source := rand.NewSource(time.Now().UnixNano() + int64(seed))
	r := rand.New(source)
	return r.Intn(n)
}

func IntnSeq(n int, len int) (res []int) {
	for i := 0; i < len; i++ {
		res = append(res, intnSeed(n, i))
	}

	return res
}

func generateChar(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seq := IntnSeq(len(charset), length)

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = charset[seq[i]]
	}
	return string(result)
}

func intn(n int) int {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	return r.Intn(n)
}

func getRandomInt(min int, max int) int {
	return intn(max-min) + min
}

func generateApiKey(uid uint32) string {
	// return 64-bit hash
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d-%s", uid, generateChar(getRandomInt(720, 1024)))))
	salt := hex.EncodeToString(hash[:])
	key := fmt.Sprintf("mk-%s", salt[:64]) // 64 bytes
	return key
}
