package proto

import (
	"context"
)

const (
	DEV_ACTION_INVAILD = 0
	DEV_ACTION_VOD     = 1 << 0
	DEV_ACTION_TUTK    = 1 << 1
	DEV_ACTION_STATUS  = 1 << 2
	DEV_ACTION_RPC     = 1 << 3
)

var gDevActions = map[string]uint64{
	"linking:vod":    DEV_ACTION_VOD,
	"linking:tutk":   DEV_ACTION_TUTK,
	"linking:status": DEV_ACTION_STATUS,
	"linking:rpc":    DEV_ACTION_RPC,
	"":               DEV_ACTION_INVAILD,
}

type DeviceAccessInfo struct {
	Uid    uint32    `json:"uid"`
	Appid  string    `json:"appid,omitempty"`
	App    string    `json:"app"`
	Device string    `json:"device"`
	Key    DeviceKey `json:"key"`
}

type DeviceKey struct {
	DeviceAccessKey string `json:"accessKey"`
	DeviceSecretKey string `json:"secretKey"`
	State           int32  `json:"state"`
}

type DeviceInterface interface {
	GetDeviceAccessInfo(ctx context.Context, accessKey string) (ret DeviceAccessInfo, err error)
}

func GetDevAction(s string) uint64 {
	if r, ok := gDevActions[s]; ok {
		return r
	}
	return DEV_ACTION_INVAILD
}
