package handler

import (
	"fmt"
	"online-server/dto"
	"online-server/model"
	"online-server/service"
	"online-server/utils"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func SeedTestData(c *gin.Context) {
	var req dto.SeedTestDataRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	if req.CaseCount <= 0 {
		req.CaseCount = 3
	}
	if req.Accounts <= 0 {
		req.Accounts = 1000
	}
	if req.Transactions <= 0 {
		req.Transactions = 500
	}
	if req.CoinType == "" {
		req.CoinType = "ETH"
	}
	now := time.Now().Unix()
	job := model.Job{
		JobNo: service.NewBusinessNo("JOB"), Type: "seed_test_data",
		Status: "running", Total: req.Accounts + req.Transactions + req.CaseCount,
		CreatedBy: c.GetString("Username"), StartedAt: &now,
	}
	_ = utils.GetDB().Create(&job).Error

	if req.Users {
		seedUser("test_officer", "test_officer@example.com", model.RoleOfficer)
		seedUser("test_auditor", "test_auditor@example.com", model.RoleAuditor)
	}

	var cases []model.Case
	for i := 1; i <= req.CaseCount; i++ {
		cs := model.Case{
			CaseNo: fmt.Sprintf("CASE-TEST-%03d", i), Name: fmt.Sprintf("测试案件%03d", i),
			Description: "测试数据生成的案件", Status: model.CaseStatusActive,
			CreatedBy: c.GetString("Username"),
		}
		utils.GetDB().Where("case_no = ?", cs.CaseNo).FirstOrCreate(&cs)
		cases = append(cases, cs)
	}

	success := 0
	for i := 1; i <= req.Accounts; i++ {
		cs := cases[(i-1)%len(cases)]
		addr := fmt.Sprintf("0x%040x", i+100000)
		acct := model.Account{
			Address: addr, CoinType: req.CoinType, AccountType: "seized_original",
			Balance: fmt.Sprintf("%d.000000000000000000", i%100+1), BalanceSource: "test",
			CaseID: &cs.ID, CaseNo: cs.CaseNo, ImportedBy: c.GetString("Username"),
			Source: "test", KeyMaterialHint: "none", Description: "测试账户",
		}
		if err := utils.GetDB().Where("address = ?", acct.Address).FirstOrCreate(&acct).Error; err == nil {
			success++
		}
	}

	for i := 1; i <= req.Transactions; i++ {
		cs := cases[(i-1)%len(cases)]
		tx := model.Transaction{
			TxNo: service.NewBusinessNo("TX"), CaseID: &cs.ID, CaseNo: cs.CaseNo,
			TxType: "test", FromAddress: fmt.Sprintf("0x%040x", i+100000),
			ToAddress: fmt.Sprintf("0x%040x", i+200000), Value: "0.01 ETH",
			CoinType: req.CoinType, Reason: "测试交易", MessageHash: fmt.Sprintf("test-message-hash-%d-%d", time.Now().UnixNano(), i),
			Status: model.StatusDraft, CreatedBy: c.GetString("Username"),
		}
		if err := utils.GetDB().Create(&tx).Error; err == nil {
			success++
		}
	}

	finished := time.Now().Unix()
	job.Status = "success"
	job.Success = success
	job.Progress = 100
	job.FinishedAt = &finished
	_ = utils.GetDB().Save(&job).Error
	service.AuditAction(c, "test_data.seed", "job", fmt.Sprint(job.ID), "", "success", "", req)
	utils.ResponseWithData(c, "测试数据生成成功", job)
}

func ClearTestData(c *gin.Context) {
	utils.GetDB().Where("source = ? OR address LIKE ?", "test", "0x000000000000000000000000000000000001%").Delete(&model.Account{})
	utils.GetDB().Where("tx_type = ?", "test").Delete(&model.Transaction{})
	utils.GetDB().Where("case_no LIKE ?", "CASE-TEST-%").Delete(&model.Case{})
	service.AuditAction(c, "test_data.clear", "test_data", "", "", "success", "", nil)
	utils.ResponseWithSuccess(c, "测试数据清理成功")
}

func TestDataSummary(c *gin.Context) {
	var cases, accounts, txs int64
	utils.GetDB().Model(&model.Case{}).Where("case_no LIKE ?", "CASE-TEST-%").Count(&cases)
	utils.GetDB().Model(&model.Account{}).Where("source = ?", "test").Count(&accounts)
	utils.GetDB().Model(&model.Transaction{}).Where("tx_type = ?", "test").Count(&txs)
	utils.ResponseWithData(c, "查询测试数据统计成功", gin.H{"cases": cases, "accounts": accounts, "transactions": txs})
}

func TestAccountTemplate(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.String(200, "address,coinType,accountType,balance,caseNo,keyMaterialHint,description\n0x0000000000000000000000000000000000000001,ETH,seized_original,1.0,CASE-TEST-001,none,测试账户\n")
}

func TestTransactionTemplate(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.String(200, "caseNo,txType,fromAddress,toAddress,value,coinType,reason\nCASE-TEST-001,test,0x0000000000000000000000000000000000000001,0x0000000000000000000000000000000000000002,0.01,ETH,测试交易\n")
}

func seedUser(username, email string, role model.Role) {
	var count int64
	utils.GetDB().Model(&model.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return
	}
	password, _ := utils.HashPassword("officer123")
	user := model.User{Username: username, Email: email, Password: password, Role: role, Status: "active"}
	_ = utils.GetDB().Create(&user).Error
}

func ListJobs(c *gin.Context) {
	var jobs []model.Job
	utils.GetDB().Order("created_at DESC").Find(&jobs)
	utils.ResponseWithData(c, "查询批量任务成功", jobs)
}

func GetJob(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var job model.Job
	if err := utils.GetDB().First(&job, id).Error; err != nil {
		utils.ResponseWithError(c, 404, "任务不存在")
		return
	}
	utils.ResponseWithData(c, "查询批量任务成功", job)
}

func JobResult(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	utils.ResponseWithData(c, "任务结果查询成功", gin.H{"jobId": id, "message": "当前任务无独立结果文件"})
}
