package models

import (
	"encoding/json"
	"github.com/TeaOSLab/EdgeAPI/internal/configs"
	"github.com/TeaOSLab/EdgeAPI/internal/errors"
	"github.com/TeaOSLab/EdgeCommon/pkg/rpc/pb"
	_ "github.com/go-sql-driver/mysql"
	"github.com/iwind/TeaGo/Tea"
	"github.com/iwind/TeaGo/dbs"
	"github.com/iwind/TeaGo/lists"
	"github.com/iwind/TeaGo/logs"
	"github.com/iwind/TeaGo/types"
	timeutil "github.com/iwind/TeaGo/utils/time"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HTTPAccessLogDAO dbs.DAO

var SharedHTTPAccessLogDAO *HTTPAccessLogDAO

func init() {
	dbs.OnReady(func() {
		SharedHTTPAccessLogDAO = NewHTTPAccessLogDAO()
	})
}

func NewHTTPAccessLogDAO() *HTTPAccessLogDAO {
	return dbs.NewDAO(&HTTPAccessLogDAO{
		DAOObject: dbs.DAOObject{
			DB:     Tea.Env,
			Table:  "edgeHTTPAccessLogs",
			Model:  new(HTTPAccessLog),
			PkName: "id",
		},
	}).(*HTTPAccessLogDAO)
}

// 创建访问日志
func (this *HTTPAccessLogDAO) CreateHTTPAccessLogs(tx *dbs.Tx, accessLogs []*pb.HTTPAccessLog) error {
	dao := randomAccessLogDAO()
	if dao == nil {
		dao = &HTTPAccessLogDAOWrapper{
			DAO:    SharedHTTPAccessLogDAO,
			NodeId: 0,
		}
	}
	return this.CreateHTTPAccessLogsWithDAO(tx, dao, accessLogs)
}

// 使用特定的DAO创建访问日志
func (this *HTTPAccessLogDAO) CreateHTTPAccessLogsWithDAO(tx *dbs.Tx, daoWrapper *HTTPAccessLogDAOWrapper, accessLogs []*pb.HTTPAccessLog) error {
	if daoWrapper == nil {
		return errors.New("dao should not be nil")
	}
	if len(accessLogs) == 0 {
		return nil
	}

	dao := daoWrapper.DAO

	// TODO 改成事务批量提交，以加快速度

	for _, accessLog := range accessLogs {
		day := timeutil.Format("Ymd", time.Unix(accessLog.Timestamp, 0))
		table, err := findAccessLogTable(dao.Instance, day, false)
		if err != nil {
			return err
		}

		fields := map[string]interface{}{}
		fields["serverId"] = accessLog.ServerId
		fields["nodeId"] = accessLog.NodeId
		fields["status"] = accessLog.Status
		fields["createdAt"] = accessLog.Timestamp
		fields["requestId"] = accessLog.RequestId + strconv.FormatInt(time.Now().UnixNano(), 10) + configs.PaddingId
		fields["firewallPolicyId"] = accessLog.FirewallPolicyId
		fields["firewallRuleGroupId"] = accessLog.FirewallRuleGroupId
		fields["firewallRuleSetId"] = accessLog.FirewallRuleSetId
		fields["firewallRuleId"] = accessLog.FirewallRuleId

		content, err := json.Marshal(accessLog)
		if err != nil {
			return err
		}
		fields["content"] = content

		_, err = dao.Query(tx).
			Table(table).
			Sets(fields).
			Insert()
		if err != nil {
			// 是否为 Error 1146: Table 'xxx.xxx' doesn't exist  如果是，则创建表之后重试
			if strings.Contains(err.Error(), "1146") {
				table, err = findAccessLogTable(dao.Instance, day, true)
				if err != nil {
					return err
				}
				_, err = dao.Query(tx).
					Table(table).
					Sets(fields).
					Insert()
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// 读取往前的 单页访问日志
func (this *HTTPAccessLogDAO) ListAccessLogs(tx *dbs.Tx, lastRequestId string, size int64, day string, serverId int64, reverse bool, hasError bool, firewallPolicyId int64, firewallRuleGroupId int64, firewallRuleSetId int64, hasFirewallPolicy bool, userId int64) (result []*HTTPAccessLog, nextLastRequestId string, hasMore bool, err error) {
	if len(day) != 8 {
		return
	}

	// 限制能查询的最大条数，防止占用内存过多
	if size > 1000 {
		size = 1000
	}

	result, nextLastRequestId, err = this.listAccessLogs(tx, lastRequestId, size, day, serverId, reverse, hasError, firewallPolicyId, firewallRuleGroupId, firewallRuleSetId, hasFirewallPolicy, userId)
	if err != nil || int64(len(result)) < size {
		return
	}

	moreResult, _, _ := this.listAccessLogs(tx, nextLastRequestId, 1, day, serverId, reverse, hasError, firewallPolicyId, firewallRuleGroupId, firewallRuleSetId, hasFirewallPolicy, userId)
	hasMore = len(moreResult) > 0
	return
}

// 读取往前的单页访问日志
func (this *HTTPAccessLogDAO) listAccessLogs(tx *dbs.Tx, lastRequestId string, size int64, day string, serverId int64, reverse bool, hasError bool, firewallPolicyId int64, firewallRuleGroupId int64, firewallRuleSetId int64, hasFirewallPolicy bool, userId int64) (result []*HTTPAccessLog, nextLastRequestId string, err error) {
	if size <= 0 {
		return nil, lastRequestId, nil
	}

	serverIds := []int64{}
	if userId > 0 {
		serverIds, err = SharedServerDAO.FindAllEnabledServerIdsWithUserId(tx, userId)
		if err != nil {
			return
		}
		if len(serverIds) == 0 {
			return
		}
	}

	accessLogLocker.RLock()
	daoList := []*HTTPAccessLogDAOWrapper{}
	for _, daoWrapper := range accessLogDAOMapping {
		daoList = append(daoList, daoWrapper)
	}
	accessLogLocker.RUnlock()

	if len(daoList) == 0 {
		daoList = []*HTTPAccessLogDAOWrapper{{
			DAO:    SharedHTTPAccessLogDAO,
			NodeId: 0,
		}}
	}

	locker := sync.Mutex{}

	count := len(daoList)
	wg := &sync.WaitGroup{}
	wg.Add(count)
	for _, daoWrapper := range daoList {
		go func(daoWrapper *HTTPAccessLogDAOWrapper) {
			defer wg.Done()

			dao := daoWrapper.DAO

			tableName, exists, err := findAccessLogTableName(dao.Instance, day)
			if !exists {
				// 表格不存在则跳过
				return
			}
			if err != nil {
				logs.Println("[DB_NODE]" + err.Error())
				return
			}

			query := dao.Query(tx)

			// 条件
			if serverId > 0 {
				query.Attr("serverId", serverId)
			} else if userId > 0 && len(serverIds) > 0 {
				query.Attr("serverId", serverIds).
					Reuse(false)
			}
			if hasError {
				query.Where("status>=400")
			}
			if firewallPolicyId > 0 {
				query.Attr("firewallPolicyId", firewallPolicyId)
			}
			if firewallRuleGroupId > 0 {
				query.Attr("firewallRuleGroupId", firewallRuleGroupId)
			}
			if firewallRuleSetId > 0 {
				query.Attr("firewallRuleSetId", firewallRuleSetId)
			}
			if hasFirewallPolicy {
				query.Where("firewallPolicyId>0")
			}

			// offset
			if len(lastRequestId) > 0 {
				if !reverse {
					query.Where("requestId<:requestId").
						Param("requestId", lastRequestId)
				} else {
					query.Where("requestId>:requestId").
						Param("requestId", lastRequestId)
				}
			}

			if !reverse {
				query.Desc("requestId")
			} else {
				query.Asc("requestId")
			}

			// 开始查询
			ones, err := query.
				Table(tableName).
				Limit(size).
				FindAll()
			if err != nil {
				logs.Println("[DB_NODE]" + err.Error())
				return
			}
			locker.Lock()
			for _, one := range ones {
				accessLog := one.(*HTTPAccessLog)
				result = append(result, accessLog)
			}
			locker.Unlock()
		}(daoWrapper)
	}
	wg.Wait()

	if len(result) == 0 {
		return nil, lastRequestId, nil
	}

	// 按照requestId排序
	sort.Slice(result, func(i, j int) bool {
		if !reverse {
			return result[i].RequestId > result[j].RequestId
		} else {
			return result[i].RequestId < result[j].RequestId
		}
	})

	if int64(len(result)) > size {
		result = result[:size]
	}

	requestId := result[len(result)-1].RequestId
	if reverse {
		lists.Reverse(result)
	}

	if !reverse {
		return result, requestId, nil
	} else {
		return result, requestId, nil
	}
}

// 根据请求ID获取访问日志
func (this *HTTPAccessLogDAO) FindAccessLogWithRequestId(tx *dbs.Tx, requestId string) (*HTTPAccessLog, error) {
	if !regexp.MustCompile(`^\d{30,}`).MatchString(requestId) {
		return nil, errors.New("invalid requestId")
	}

	accessLogLocker.RLock()
	daoList := []*HTTPAccessLogDAOWrapper{}
	for _, daoWrapper := range accessLogDAOMapping {
		daoList = append(daoList, daoWrapper)
	}
	accessLogLocker.RUnlock()

	if len(daoList) == 0 {
		daoList = []*HTTPAccessLogDAOWrapper{{
			DAO:    SharedHTTPAccessLogDAO,
			NodeId: 0,
		}}
	}

	count := len(daoList)
	wg := &sync.WaitGroup{}
	wg.Add(count)
	var result *HTTPAccessLog = nil
	day := timeutil.FormatTime("Ymd", types.Int64(requestId[:10]))
	for _, daoWrapper := range daoList {
		go func(daoWrapper *HTTPAccessLogDAOWrapper) {
			defer wg.Done()

			dao := daoWrapper.DAO

			tableName, exists, err := findAccessLogTableName(dao.Instance, day)
			if err != nil {
				logs.Println("[DB_NODE]" + err.Error())
				return
			}
			if !exists {
				return
			}

			one, err := dao.Query(tx).
				Table(tableName).
				Attr("requestId", requestId).
				Find()
			if err != nil {
				logs.Println("[DB_NODE]" + err.Error())
				return
			}
			if one != nil {
				result = one.(*HTTPAccessLog)
			}
		}(daoWrapper)
	}
	wg.Wait()
	return result, nil
}
