package models

import (
	"encoding/json"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	_ "github.com/go-sql-driver/mysql"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/dbs"
	"github.com/iwind/TeaGo/maps"
	"github.com/iwind/TeaGo/types"
)

const (
	ReverseProxyStateEnabled  = 1 // 已启用
	ReverseProxyStateDisabled = 0 // 已禁用
)

type ReverseProxyDAO dbs.DAO

func NewReverseProxyDAO() *ReverseProxyDAO {
	return dbs.NewDAO(&ReverseProxyDAO{
		DAOObject: dbs.DAOObject{
			DB:     Tea.Env,
			Table:  "edgeReverseProxies",
			Model:  new(ReverseProxy),
			PkName: "id",
		},
	}).(*ReverseProxyDAO)
}

var SharedReverseProxyDAO *ReverseProxyDAO

func init() {
	dbs.OnReady(func() {
		SharedReverseProxyDAO = NewReverseProxyDAO()
	})
}

// 初始化
func (this *ReverseProxyDAO) Init() {
	_ = this.DAOObject.Init()
}

// 启用条目
func (this *ReverseProxyDAO) EnableReverseProxy(tx *dbs.Tx, id int64) error {
	_, err := this.Query(tx).
		Pk(id).
		Set("state", ReverseProxyStateEnabled).
		Update()
	if err != nil {
		return err
	}
	return this.NotifyUpdate(tx, id)
}

// 禁用条目
func (this *ReverseProxyDAO) DisableReverseProxy(tx *dbs.Tx, id int64) error {
	_, err := this.Query(tx).
		Pk(id).
		Set("state", ReverseProxyStateDisabled).
		Update()
	if err != nil {
		return err
	}
	return this.NotifyUpdate(tx, id)
}

// 查找启用中的条目
func (this *ReverseProxyDAO) FindEnabledReverseProxy(tx *dbs.Tx, id int64) (*ReverseProxy, error) {
	result, err := this.Query(tx).
		Pk(id).
		Attr("state", ReverseProxyStateEnabled).
		Find()
	if result == nil {
		return nil, err
	}
	return result.(*ReverseProxy), err
}

// 根据iD组合配置
func (this *ReverseProxyDAO) ComposeReverseProxyConfig(tx *dbs.Tx, reverseProxyId int64) (*serverconfigs.ReverseProxyConfig, error) {
	reverseProxy, err := this.FindEnabledReverseProxy(tx, reverseProxyId)
	if err != nil {
		return nil, err
	}
	if reverseProxy == nil {
		return nil, nil
	}

	config := &serverconfigs.ReverseProxyConfig{}
	config.Id = int64(reverseProxy.Id)
	config.IsOn = reverseProxy.IsOn == 1
	config.RequestHostType = types.Int8(reverseProxy.RequestHostType)
	config.RequestHost = reverseProxy.RequestHost
	config.RequestURI = reverseProxy.RequestURI
	config.StripPrefix = reverseProxy.StripPrefix
	config.AutoFlush = reverseProxy.AutoFlush == 1

	schedulingConfig := &serverconfigs.SchedulingConfig{}
	if len(reverseProxy.Scheduling) > 0 && reverseProxy.Scheduling != "null" {
		err = json.Unmarshal([]byte(reverseProxy.Scheduling), schedulingConfig)
		if err != nil {
			return nil, err
		}
		config.Scheduling = schedulingConfig
	}
	if len(reverseProxy.PrimaryOrigins) > 0 && reverseProxy.PrimaryOrigins != "null" {
		originRefs := []*serverconfigs.OriginRef{}
		err = json.Unmarshal([]byte(reverseProxy.PrimaryOrigins), &originRefs)
		if err != nil {
			return nil, err
		}
		for _, ref := range originRefs {
			originConfig, err := SharedOriginDAO.ComposeOriginConfig(tx, ref.OriginId)
			if err != nil {
				return nil, err
			}
			if originConfig != nil {
				config.AddPrimaryOrigin(originConfig)
			}
		}
	}

	if len(reverseProxy.BackupOrigins) > 0 && reverseProxy.BackupOrigins != "null" {
		originRefs := []*serverconfigs.OriginRef{}
		err = json.Unmarshal([]byte(reverseProxy.BackupOrigins), &originRefs)
		if err != nil {
			return nil, err
		}
		for _, originConfig := range originRefs {
			originConfig, err := SharedOriginDAO.ComposeOriginConfig(tx, originConfig.OriginId)
			if err != nil {
				return nil, err
			}
			if originConfig != nil {
				config.AddBackupOrigin(originConfig)
			}
		}
	}

	// add headers
	if IsNotNull(reverseProxy.AddHeaders) {
		addHeaders := []string{}
		err = json.Unmarshal([]byte(reverseProxy.AddHeaders), &addHeaders)
		if err != nil {
			return nil, err
		}
		config.AddHeaders = addHeaders
	}

	return config, nil
}

// 创建反向代理
func (this *ReverseProxyDAO) CreateReverseProxy(tx *dbs.Tx, adminId int64, userId int64, schedulingJSON []byte, primaryOriginsJSON []byte, backupOriginsJSON []byte) (int64, error) {
	op := NewReverseProxyOperator()
	op.IsOn = true
	op.State = ReverseProxyStateEnabled
	op.AdminId = adminId
	op.UserId = userId
	op.AddHeaders = "[\"X-Real-IP\"]"

	if len(schedulingJSON) > 0 {
		op.Scheduling = string(schedulingJSON)
	}
	if len(primaryOriginsJSON) > 0 {
		op.PrimaryOrigins = string(primaryOriginsJSON)
	}
	if len(backupOriginsJSON) > 0 {
		op.BackupOrigins = string(backupOriginsJSON)
	}
	err := this.Save(tx, op)
	if err != nil {
		return 0, err
	}

	return types.Int64(op.Id), nil
}

// 修改反向代理调度算法
func (this *ReverseProxyDAO) UpdateReverseProxyScheduling(tx *dbs.Tx, reverseProxyId int64, schedulingJSON []byte) error {
	if reverseProxyId <= 0 {
		return errors.New("invalid reverseProxyId")
	}
	op := NewReverseProxyOperator()
	op.Id = reverseProxyId
	if len(schedulingJSON) > 0 {
		op.Scheduling = string(schedulingJSON)
	} else {
		op.Scheduling = "null"
	}
	err := this.Save(tx, op)
	if err != nil {
		return err
	}
	return this.NotifyUpdate(tx, reverseProxyId)
}

// 修改主要源站
func (this *ReverseProxyDAO) UpdateReverseProxyPrimaryOrigins(tx *dbs.Tx, reverseProxyId int64, origins []byte) error {
	if reverseProxyId <= 0 {
		return errors.New("invalid reverseProxyId")
	}
	op := NewReverseProxyOperator()
	op.Id = reverseProxyId
	if len(origins) > 0 {
		op.PrimaryOrigins = origins
	} else {
		op.PrimaryOrigins = "[]"
	}
	err := this.Save(tx, op)
	if err != nil {
		return err
	}
	return this.NotifyUpdate(tx, reverseProxyId)
}

// 修改备用源站
func (this *ReverseProxyDAO) UpdateReverseProxyBackupOrigins(tx *dbs.Tx, reverseProxyId int64, origins []byte) error {
	if reverseProxyId <= 0 {
		return errors.New("invalid reverseProxyId")
	}
	op := NewReverseProxyOperator()
	op.Id = reverseProxyId
	if len(origins) > 0 {
		op.BackupOrigins = origins
	} else {
		op.BackupOrigins = "[]"
	}
	err := this.Save(tx, op)
	if err != nil {
		return err
	}
	return this.NotifyUpdate(tx, reverseProxyId)
}

// 修改是否启用
func (this *ReverseProxyDAO) UpdateReverseProxy(tx *dbs.Tx, reverseProxyId int64, requestHostType int8, requestHost string, requestURI string, stripPrefix string, autoFlush bool, addHeaders []string) error {
	if reverseProxyId <= 0 {
		return errors.New("invalid reverseProxyId")
	}

	op := NewReverseProxyOperator()
	op.Id = reverseProxyId

	if requestHostType < 0 {
		requestHostType = 0
	}
	op.RequestHostType = requestHostType

	op.RequestHost = requestHost
	op.RequestURI = requestURI
	op.StripPrefix = stripPrefix
	op.AutoFlush = autoFlush

	if len(addHeaders) == 0 {
		addHeaders = []string{}
	}
	addHeadersJSON, err := json.Marshal(addHeaders)
	if err != nil {
		return err
	}
	op.AddHeaders = addHeadersJSON

	err = this.Save(tx, op)
	if err != nil {
		return err
	}
	return this.NotifyUpdate(tx, reverseProxyId)
}

// 查找包含某个源站的反向代理ID
func (this *ReverseProxyDAO) FindReverseProxyContainsOriginId(tx *dbs.Tx, originId int64) (int64, error) {
	return this.Query(tx).
		ResultPk().
		Where("(JSON_CONTAINS(primaryOrigins, :jsonQuery) OR JSON_CONTAINS(backupOrigins, :jsonQuery))").
		Param("jsonQuery", maps.Map{
			"originId": originId,
		}.AsJSON()).
		FindInt64Col(0)
}

// 通知更新
func (this *ReverseProxyDAO) NotifyUpdate(tx *dbs.Tx, reverseProxyId int64) error {
	serverId, err := SharedServerDAO.FindEnabledServerIdWithReverseProxyId(tx, reverseProxyId)
	if err != nil {
		return err
	}
	if serverId > 0 {
		return SharedServerDAO.NotifyUpdate(tx, serverId)
	}
	return nil
}
