package models

import (
	"encoding/json"
	"errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs"
	"github.com/TeaOSLab/EdgeCommon/pkg/serverconfigs/shared"
	_ "github.com/go-sql-driver/mysql"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/dbs"
	"github.com/iwind/TeaGo/types"
)

const (
	HTTPGzipStateEnabled  = 1 // 已启用
	HTTPGzipStateDisabled = 0 // 已禁用
)

type HTTPGzipDAO dbs.DAO

func NewHTTPGzipDAO() *HTTPGzipDAO {
	return dbs.NewDAO(&HTTPGzipDAO{
		DAOObject: dbs.DAOObject{
			DB:     Tea.Env,
			Table:  "edgeHTTPGzips",
			Model:  new(HTTPGzip),
			PkName: "id",
		},
	}).(*HTTPGzipDAO)
}

var SharedHTTPGzipDAO = NewHTTPGzipDAO()

// 启用条目
func (this *HTTPGzipDAO) EnableHTTPGzip(id int64) error {
	_, err := this.Query().
		Pk(id).
		Set("state", HTTPGzipStateEnabled).
		Update()
	return err
}

// 禁用条目
func (this *HTTPGzipDAO) DisableHTTPGzip(id int64) error {
	_, err := this.Query().
		Pk(id).
		Set("state", HTTPGzipStateDisabled).
		Update()
	return err
}

// 查找启用中的条目
func (this *HTTPGzipDAO) FindEnabledHTTPGzip(id int64) (*HTTPGzip, error) {
	result, err := this.Query().
		Pk(id).
		Attr("state", HTTPGzipStateEnabled).
		Find()
	if result == nil {
		return nil, err
	}
	return result.(*HTTPGzip), err
}

// 组合配置
func (this *HTTPGzipDAO) ComposeGzipConfig(gzipId int64) (*serverconfigs.HTTPGzipConfig, error) {
	gzip, err := this.FindEnabledHTTPGzip(gzipId)
	if err != nil {
		return nil, err
	}

	if gzip == nil {
		return nil, nil
	}

	config := &serverconfigs.HTTPGzipConfig{}
	config.Id = int64(gzip.Id)
	config.IsOn = gzip.IsOn == 1
	if len(gzip.MinLength) > 0 && gzip.MinLength != "null" {
		minLengthConfig := &shared.SizeCapacity{}
		err = json.Unmarshal([]byte(gzip.MinLength), minLengthConfig)
		if err != nil {
			return nil, err
		}
		config.MinLength = minLengthConfig
	}
	if len(gzip.MaxLength) > 0 && gzip.MaxLength != "null" {
		maxLengthConfig := &shared.SizeCapacity{}
		err = json.Unmarshal([]byte(gzip.MaxLength), maxLengthConfig)
		if err != nil {
			return nil, err
		}
		config.MaxLength = maxLengthConfig
	}
	config.Level = types.Int8(gzip.Level)
	return config, nil
}

// 创建Gzip
func (this *HTTPGzipDAO) CreateGzip(level int, minLengthJSON []byte, maxLengthJSON []byte) (int64, error) {
	op := NewHTTPGzipOperator()
	op.State = HTTPGzipStateEnabled
	op.IsOn = true
	op.Level = level
	if len(minLengthJSON) > 0 {
		op.MinLength = string(minLengthJSON)
	}
	if len(maxLengthJSON) > 0 {
		op.MaxLength = string(maxLengthJSON)
	}
	_, err := this.Save(op)
	if err != nil {
		return 0, err
	}
	return types.Int64(op.Id), nil
}

// 修改Gzip
func (this *HTTPGzipDAO) UpdateGzip(gzipId int64, level int, minLengthJSON []byte, maxLengthJSON []byte) error {
	if gzipId <= 0 {
		return errors.New("invalid gzipId")
	}
	op := NewHTTPGzipOperator()
	op.Id = gzipId
	op.Level = level
	if len(minLengthJSON) > 0 {
		op.MinLength = string(minLengthJSON)
	}
	if len(maxLengthJSON) > 0 {
		op.MaxLength = string(maxLengthJSON)
	}
	_, err := this.Save(op)
	return err
}