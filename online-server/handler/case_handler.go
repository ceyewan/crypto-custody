package handler

import (
	"net/http"
	"online-server/dto"
	"online-server/model"
	"online-server/service"
	"online-server/utils"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func CreateCase(c *gin.Context) {
	var req dto.CaseRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	status := model.CaseStatusActive
	if req.Status != "" {
		status = model.CaseStatus(req.Status)
	}
	caseNo := strings.TrimSpace(req.CaseNo)
	if caseNo == "" {
		generated, err := service.NewCaseService().GenerateCaseNo()
		if err != nil {
			utils.ResponseWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
		caseNo = generated
	}
	cs := model.Case{
		CaseNo:      caseNo,
		Name:        req.Name,
		Description: req.Description,
		Status:      status,
		CreatedBy:   c.GetString("Username"),
	}
	if err := utils.GetDB().Create(&cs).Error; err != nil {
		service.AuditAction(c, "case.create", "case", "", caseNo, "failure", err.Error(), nil)
		utils.ResponseWithError(c, http.StatusBadRequest, "创建案件失败: "+err.Error())
		return
	}
	service.AuditAction(c, "case.create", "case", strconv.FormatUint(uint64(cs.ID), 10), cs.CaseNo, "success", "", cs)
	utils.ResponseWithData(c, "案件创建成功", cs)
}

func ListCases(c *gin.Context) {
	var req dto.CaseListRequest
	_ = c.ShouldBindQuery(&req)
	cases, total, err := service.NewCaseService().List(req.Page, req.PageSize, req.CaseNo, req.Keyword, req.Status)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	utils.ResponseWithData(c, "查询案件列表成功", gin.H{
		"items": cases, "total": total, "page": req.Page, "pageSize": req.PageSize,
	})
}

func GetCase(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var cs model.Case
	if err := utils.GetDB().First(&cs, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "案件不存在")
		return
	}
	utils.ResponseWithData(c, "查询案件成功", cs)
}

func UpdateCase(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.CaseRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	var cs model.Case
	if err := utils.GetDB().First(&cs, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "案件不存在")
		return
	}
	if req.CaseNo != "" {
		cs.CaseNo = req.CaseNo
	}
	if req.Name != "" {
		cs.Name = req.Name
	}
	cs.Description = req.Description
	if req.Status != "" {
		cs.Status = model.CaseStatus(req.Status)
	}
	if err := utils.GetDB().Save(&cs).Error; err != nil {
		service.AuditAction(c, "case.update", "case", strconv.FormatUint(uint64(cs.ID), 10), cs.CaseNo, "failure", err.Error(), nil)
		utils.ResponseWithError(c, http.StatusBadRequest, "更新案件失败: "+err.Error())
		return
	}
	service.AuditAction(c, "case.update", "case", strconv.FormatUint(uint64(cs.ID), 10), cs.CaseNo, "success", "", cs)
	utils.ResponseWithData(c, "案件更新成功", cs)
}

func DeleteCase(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var cs model.Case
	if err := utils.GetDB().First(&cs, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "案件不存在")
		return
	}
	if err := utils.GetDB().Delete(&cs).Error; err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "删除案件失败: "+err.Error())
		return
	}
	service.AuditAction(c, "case.delete", "case", strconv.FormatUint(uint64(cs.ID), 10), cs.CaseNo, "success", "", nil)
	utils.ResponseWithSuccess(c, "案件删除成功")
}

func GetCaseAccounts(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var cs model.Case
	if err := utils.GetDB().First(&cs, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "案件不存在")
		return
	}
	var accounts []model.Account
	if err := utils.GetDB().Where("case_id = ? OR case_no = ?", cs.ID, cs.CaseNo).Order("created_at DESC").Find(&accounts).Error; err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "查询案件账户失败: "+err.Error())
		return
	}
	utils.ResponseWithData(c, "查询案件账户成功", accounts)
}

func LinkCaseAccount(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.LinkAccountRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	var cs model.Case
	if err := utils.GetDB().First(&cs, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "案件不存在")
		return
	}
	var account model.Account
	if err := utils.GetDB().First(&account, req.AccountID).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "账户不存在")
		return
	}
	account.CaseID = &cs.ID
	account.CaseNo = cs.CaseNo
	if err := utils.GetDB().Save(&account).Error; err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "关联账户失败: "+err.Error())
		return
	}
	service.AuditAction(c, "case.link_account", "account", strconv.FormatUint(uint64(account.ID), 10), cs.CaseNo, "success", "", account)
	utils.ResponseWithData(c, "账户关联成功", account)
}

func UnlinkCaseAccount(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	accountID, err := strconv.ParseUint(c.Param("accountId"), 10, 32)
	if err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, "无效的账户ID")
		return
	}
	var cs model.Case
	if err := utils.GetDB().First(&cs, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "案件不存在")
		return
	}
	if err := utils.GetDB().Model(&model.Account{}).Where("id = ?", uint(accountID)).Updates(map[string]interface{}{"case_id": nil, "case_no": ""}).Error; err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "解除关联失败: "+err.Error())
		return
	}
	service.AuditAction(c, "case.unlink_account", "account", c.Param("accountId"), cs.CaseNo, "success", "", nil)
	utils.ResponseWithSuccess(c, "账户解除关联成功")
}

func ImportCustodyWalletResult(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.CustodyWalletResultRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	var cs model.Case
	if err := utils.GetDB().First(&cs, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "案件不存在")
		return
	}
	coinType := req.CoinType
	if coinType == "" {
		coinType = "ETH"
	}
	account := model.Account{
		Address:         req.CustodyAddress,
		CoinType:        coinType,
		AccountType:     "custody_wallet",
		Balance:         "0",
		BalanceSource:   "chain",
		CaseID:          &cs.ID,
		CaseNo:          cs.CaseNo,
		ImportedBy:      c.GetString("Username"),
		Source:          "offline_result",
		KeyMaterialHint: "offline_generated",
		OfflineRefNo:    req.OfflineRefNo,
		Description:     "离线系统生成的案件托管钱包",
	}
	if err := utils.GetDB().Where("address = ?", account.Address).FirstOrCreate(&account).Error; err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, "导入托管钱包失败: "+err.Error())
		return
	}
	cs.CustodyAccountID = &account.ID
	cs.CustodyAddress = account.Address
	if err := utils.GetDB().Save(&cs).Error; err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "更新案件托管钱包失败: "+err.Error())
		return
	}
	service.AuditAction(c, "case.import_custody_wallet", "case", strconv.FormatUint(uint64(cs.ID), 10), cs.CaseNo, "success", "", req)
	utils.ResponseWithData(c, "托管钱包导入成功", gin.H{"case": cs, "account": account})
}
